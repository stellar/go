package horizon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/render/hal"
)

// StreamTest utility struct to wrap SSE related tests.
type StreamTest struct {
	action       pageAction
	ledgerSource *ledger.TestingSource
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
	ctx          context.Context
}

// NewStreamTest executes an SSE related test, letting you simulate ledger closings via
// AddLedger.
func NewStreamTest(
	action pageAction,
	currentLedger uint32,
	request *http.Request,
	checkResponse func(w *httptest.ResponseRecorder),
) *StreamTest {
	s := &StreamTest{
		action:       action,
		ledgerSource: ledger.NewTestingSource(currentLedger),
		wg:           &sync.WaitGroup{},
	}
	s.ctx, s.cancel = context.WithCancel(request.Context())

	streamHandler := sse.StreamHandler{
		LedgerSource: s.ledgerSource,
	}
	handler := streamablePageHandler(s.action, streamHandler)

	s.wg.Add(1)
	go func() {
		w := httptest.NewRecorder()
		handler.renderStream(w, request.WithContext(s.ctx))
		s.wg.Done()
		s.cancel()

		checkResponse(w)
	}()

	return s
}

// AddLedger pushes a new ledger to the stream handler. AddLedger() will block until
// the new ledger has been read by the stream handler
func (s *StreamTest) AddLedger(sequence uint32) {
	s.wg.Add(1)
	defer s.wg.Done()
	s.ledgerSource.AddLedger(sequence)
}

// TryAddLedger pushes a new ledger to the stream handler. TryAddLedger() will block for at
// most 1 second until the new ledger has been read by the stream handler
func (s *StreamTest) TryAddLedger(sequence uint32) {
	s.wg.Add(1)
	defer s.wg.Done()
	s.ledgerSource.TryAddLedger(s.ctx, sequence, 2*time.Second)
}

// Wait blocks testing until the stream test has finished running.
func (s *StreamTest) Wait(expectLimitReached bool) {
	if !expectLimitReached {
		// first send a ledger to the stream handler so we can ensure that at least one
		// iteration of the stream loop has been executed
		s.TryAddLedger(0)
		s.cancel()
	}
	s.wg.Wait()
}

type testPage struct {
	Value       string `json:"value"`
	pagingToken int
}

func (p testPage) PagingToken() string {
	return fmt.Sprintf("%v", p.pagingToken)
}

type testPageAction struct {
	objects []string
	lock    sync.Mutex
}

func (action *testPageAction) appendObjects(objects ...string) {
	action.lock.Lock()
	defer action.lock.Unlock()
	action.objects = append(action.objects, objects...)
}

func (action *testPageAction) GetResourcePage(r *http.Request) ([]hal.Pageable, error) {
	action.lock.Lock()
	defer action.lock.Unlock()

	cursor := r.Header.Get("Last-Event-ID")
	if cursor == "" {
		cursor = r.URL.Query().Get("cursor")
	}
	if cursor == "" {
		cursor = "0"
	}
	parsedCursor, err := strconv.Atoi(cursor)
	if err != nil {
		return nil, err
	}

	limit := len(action.objects)
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil {
			return nil, err
		}
	}

	if parsedCursor < 0 {
		return nil, fmt.Errorf("cursor cannot be negative")
	}

	if parsedCursor >= len(action.objects) {
		return []hal.Pageable{}, nil
	}
	response := []hal.Pageable{}
	for i, object := range action.objects[parsedCursor:] {
		if len(response) >= limit {
			break
		}

		response = append(response, testPage{Value: object, pagingToken: parsedCursor + i + 1})
	}

	return response, nil
}

func streamRequest(t *testing.T, queryParams string) *http.Request {
	request, err := http.NewRequest("GET", "http://localhost?"+queryParams, nil)
	if err != nil {
		t.Fatalf("could not construct request: %v", err)
	}
	ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chi.NewRouteContext())
	request = request.WithContext(ctx)

	return request
}

func expectResponse(t *testing.T, expectedResponse []string) func(*httptest.ResponseRecorder) {
	return func(w *httptest.ResponseRecorder) {
		var response []string
		for _, line := range strings.Split(w.Body.String(), "\n") {
			if strings.HasPrefix(line, "data: {") {
				jsonString := line[len("data: "):]
				var page testPage
				err := json.Unmarshal([]byte(jsonString), &page)
				if err != nil {
					t.Fatalf("could not parse json %v", err)
				}
				response = append(response, page.Value)
			}
		}

		if len(expectedResponse) != len(response) {
			t.Fatalf("expected %v but got %v", expectedResponse, response)
		}

		for i, entry := range expectedResponse {
			if entry != response[i] {
				t.Fatalf("expected %v but got %v", expectedResponse, response)
			}
		}
	}
}

func TestRenderStream(t *testing.T) {
	action := &testPageAction{
		objects: []string{"a", "b", "c"},
	}

	t.Run("without offset", func(t *testing.T) {
		request := streamRequest(t, "")
		st := NewStreamTest(
			action,
			3,
			request,
			expectResponse(t, []string{"a", "b", "c", "d", "e", "f"}),
		)

		st.AddLedger(4)
		action.appendObjects("d", "e")

		st.AddLedger(6)
		action.appendObjects("f")

		st.Wait(false)
	})

	action.objects = []string{"a", "b", "c"}
	t.Run("with offset", func(t *testing.T) {
		request := streamRequest(t, "cursor=1")
		st := NewStreamTest(
			action,
			3,
			request,
			expectResponse(t, []string{"b", "c", "d", "e", "f"}),
		)

		st.AddLedger(4)
		action.appendObjects("d", "e")

		st.AddLedger(6)
		action.appendObjects("f")

		st.Wait(false)
	})

	action.objects = []string{"a", "b", "c"}
	t.Run("with limit", func(t *testing.T) {
		request := streamRequest(t, "limit=2")
		st := NewStreamTest(
			action,
			3,
			request,
			expectResponse(t, []string{"a", "b"}),
		)

		st.Wait(true)
	})

	action.objects = []string{"a", "b", "c", "d", "e"}
	t.Run("with limit and offset", func(t *testing.T) {
		request := streamRequest(t, "limit=2&cursor=1")
		st := NewStreamTest(
			action,
			3,
			request,
			expectResponse(t, []string{"b", "c"}),
		)

		st.Wait(true)
	})

	action.objects = []string{"a"}
	t.Run("reach limit after multiple iterations", func(t *testing.T) {
		request := streamRequest(t, "limit=3&cursor=1")
		st := NewStreamTest(
			action,
			3,
			request,
			expectResponse(t, []string{"b", "c", "d"}),
		)

		st.AddLedger(4)
		action.appendObjects("b")

		st.AddLedger(5)
		action.appendObjects("c", "d", "e", "f", "g")

		st.TryAddLedger(0)
		st.Wait(true)
	})
}
