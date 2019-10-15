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
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/render/hal"
)

// StreamTest utility struct to wrap SSE related tests.
type StreamTest struct {
	ledgerSource  *ledger.TestingSource
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
	w             *httptest.ResponseRecorder
	checkResponse func(w *httptest.ResponseRecorder)
	ctx           context.Context
}

func newStreamTest(
	handler http.HandlerFunc,
	ledgerSource *ledger.TestingSource,
	request *http.Request,
	checkResponse func(w *httptest.ResponseRecorder),
) *StreamTest {
	s := &StreamTest{
		ledgerSource:  ledgerSource,
		w:             httptest.NewRecorder(),
		checkResponse: checkResponse,
		wg:            &sync.WaitGroup{},
	}
	s.ctx, s.cancel = context.WithCancel(request.Context())

	s.wg.Add(1)
	go func() {
		handler(s.w, request.WithContext(s.ctx))
		s.wg.Done()
		s.cancel()
	}()

	return s
}

// NewstreamableObjectTest tests the SSE functionality of a pageAction
func NewStreamablePageTest(
	action *testPageAction,
	currentLedger uint32,
	request *http.Request,
	checkResponse func(w *httptest.ResponseRecorder),
) *StreamTest {
	ledgerSource := ledger.NewTestingSource(currentLedger)
	action.ledgerSource = ledgerSource
	streamHandler := sse.StreamHandler{LedgerSource: ledgerSource}
	handler := streamablePageHandler(action, streamHandler)

	return newStreamTest(
		handler.renderStream,
		ledgerSource,
		request,
		checkResponse,
	)
}

// NewstreamableObjectTest tests the SSE functionality of a streamableObjectAction
func NewstreamableObjectTest(
	action *testObjectAction,
	currentLedger uint32,
	request *http.Request,
	checkResponse func(w *httptest.ResponseRecorder),
) *StreamTest {
	ledgerSource := ledger.NewTestingSource(currentLedger)
	action.ledgerSource = ledgerSource
	streamHandler := sse.StreamHandler{LedgerSource: ledgerSource}
	handler := streamableObjectActionHandler{action, streamHandler}

	return newStreamTest(
		handler.renderStream,
		ledgerSource,
		request,
		checkResponse,
	)
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
		s.TryAddLedger(s.ledgerSource.CurrentLedger() + 1)
		s.cancel()
	}
	s.wg.Wait()
	s.checkResponse(s.w)
}

type testPage struct {
	Value       string `json:"value"`
	pagingToken int
}

func (p testPage) PagingToken() string {
	return fmt.Sprintf("%v", p.pagingToken)
}

type testPageAction struct {
	objects      map[uint32][]string
	ledgerSource ledger.Source
}

func (action *testPageAction) GetResourcePage(
	w actions.HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	objects, ok := action.objects[action.ledgerSource.CurrentLedger()]
	if !ok {
		return nil, fmt.Errorf("unexpected ledger")
	}

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

	limit := len(objects)
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil {
			return nil, err
		}
	}

	if parsedCursor < 0 {
		return nil, fmt.Errorf("cursor cannot be negative")
	}

	if parsedCursor >= len(objects) {
		return []hal.Pageable{}, nil
	}

	response := []hal.Pageable{}
	for i, object := range objects[parsedCursor:] {
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

func unmarashalPage(jsonString string) (string, error) {
	var page testPage
	err := json.Unmarshal([]byte(jsonString), &page)
	return page.Value, err
}

func expectResponse(
	t *testing.T,
	unmarshal func(string) (string, error),
	expectedResponse []string,
) func(*httptest.ResponseRecorder) {
	return func(w *httptest.ResponseRecorder) {
		var response []string
		for _, line := range strings.Split(w.Body.String(), "\n") {
			if line == "data: \"hello\"" || line == "data: \"byebye\"" {
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				jsonString := line[len("data: "):]
				value, err := unmarshal(jsonString)
				if err != nil {
					t.Fatalf("could not parse json %v", err)
				}
				response = append(response, value)
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

func TestPageStream(t *testing.T) {
	t.Run("without offset", func(t *testing.T) {
		request := streamRequest(t, "")
		action := &testPageAction{
			objects: map[uint32][]string{
				3: []string{"a", "b", "c"},
				4: []string{"a", "b", "c", "d", "e"},
				6: []string{"a", "b", "c", "d", "e", "f"},
				7: []string{"a", "b", "c", "d", "e", "f"},
			},
		}
		st := NewStreamablePageTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalPage, []string{"a", "b", "c", "d", "e", "f"}),
		)

		st.AddLedger(4)
		st.AddLedger(6)

		st.Wait(false)
	})

	t.Run("with offset", func(t *testing.T) {
		request := streamRequest(t, "cursor=1")
		action := &testPageAction{
			objects: map[uint32][]string{
				3: []string{"a", "b", "c"},
				4: []string{"a", "b", "c", "d", "e"},
				6: []string{"a", "b", "c", "d", "e", "f"},
				7: []string{"a", "b", "c", "d", "e", "f"},
			},
		}
		st := NewStreamablePageTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalPage, []string{"b", "c", "d", "e", "f"}),
		)

		st.AddLedger(4)
		st.AddLedger(6)

		st.Wait(false)
	})

	t.Run("with limit", func(t *testing.T) {
		request := streamRequest(t, "limit=2")
		action := &testPageAction{
			objects: map[uint32][]string{
				3: []string{"a", "b", "c"},
			},
		}
		st := NewStreamablePageTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalPage, []string{"a", "b"}),
		)

		st.Wait(true)
	})

	t.Run("with limit and offset", func(t *testing.T) {
		request := streamRequest(t, "limit=2&cursor=1")
		action := &testPageAction{
			objects: map[uint32][]string{
				3: []string{"a", "b", "c", "d", "e"},
			},
		}
		st := NewStreamablePageTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalPage, []string{"b", "c"}),
		)

		st.Wait(true)
	})

	t.Run("reach limit after multiple iterations", func(t *testing.T) {
		request := streamRequest(t, "limit=3&cursor=1")
		action := &testPageAction{
			objects: map[uint32][]string{
				3: []string{"a"},
				4: []string{"a", "b"},
				5: []string{"a", "b", "c", "d", "e", "f", "g"},
			},
		}
		st := NewStreamablePageTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalPage, []string{"b", "c", "d"}),
		)

		st.AddLedger(4)
		st.AddLedger(5)

		st.Wait(true)
	})
}

type stringObject string

func (s stringObject) Equals(other actions.StreamableObjectResponse) bool {
	otherString, ok := other.(stringObject)
	if !ok {
		return false
	}
	return s == otherString
}

func unmarashalString(jsonString string) (string, error) {
	var object stringObject
	err := json.Unmarshal([]byte(jsonString), &object)
	return string(object), err
}

type testObjectAction struct {
	objects      map[uint32]stringObject
	ledgerSource ledger.Source
}

func (action *testObjectAction) GetResource(
	w actions.HeaderWriter,
	r *http.Request,
) (actions.StreamableObjectResponse, error) {
	ledger := action.ledgerSource.CurrentLedger()
	object, ok := action.objects[ledger]
	if !ok {
		return nil, fmt.Errorf("unexpected ledger")
	}

	return object, nil
}

func TestObjectStream(t *testing.T) {
	t.Run("without interior duplicates", func(t *testing.T) {
		request := streamRequest(t, "")
		action := &testObjectAction{
			objects: map[uint32]stringObject{
				3: "a",
				4: "b",
				5: "c",
				6: "c",
			},
		}

		st := NewstreamableObjectTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalString, []string{"a", "b", "c"}),
		)

		st.AddLedger(4)
		st.AddLedger(5)
		st.Wait(false)
	})

	t.Run("with interior duplicates", func(t *testing.T) {
		request := streamRequest(t, "")
		action := &testObjectAction{
			objects: map[uint32]stringObject{
				3: "a",
				4: "b",
				5: "b",
				6: "c",
				7: "c",
			},
		}

		st := NewstreamableObjectTest(
			action,
			3,
			request,
			expectResponse(t, unmarashalString, []string{"a", "b", "c"}),
		)

		st.AddLedger(4)
		st.AddLedger(5)
		st.AddLedger(6)

		st.Wait(false)
	})

	t.Run("limit reached", func(t *testing.T) {
		request := streamRequest(t, "")
		action := &testObjectAction{
			objects: map[uint32]stringObject{
				1:  "a",
				2:  "b",
				3:  "b",
				4:  "c",
				5:  "d",
				6:  "e",
				7:  "f",
				8:  "g",
				9:  "h",
				10: "i",
				11: "j",
				12: "k",
			},
		}

		st := NewstreamableObjectTest(
			action,
			1,
			request,
			expectResponse(
				t,
				unmarashalString,
				[]string{
					"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
				},
			),
		)

		for i := uint32(1); i <= 11; i++ {
			st.AddLedger(i)
		}

		st.Wait(true)
	})
}
