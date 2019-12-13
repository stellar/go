package errors

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testE struct {
	statusCode int
	ErrorStr   string `json:"error"`
}

func (te testE) StatusCode() int {
	return te.statusCode
}

func (te testE) Error() string {
	return te.ErrorStr
}

// TestErrorsRender tests the render cases
func TestErrorsRender(t *testing.T) {
	errors := New("application/error+json", empty500{}, nil, log.DefaultLogger)

	testCases := []struct {
		name          string
		err           error
		errRegistered bool
		e             E
		wantBody      string
		wantCode      int
	}{
		{
			name:          "err not registered",
			err:           fmt.Errorf("an error"),
			errRegistered: false,
			e:             DefaultE,
			wantBody:      `{}`,
			wantCode:      500,
		},
		{
			name:          "err registered",
			err:           fmt.Errorf("an error"),
			errRegistered: true,
			e:             testE{statusCode: 418, ErrorStr: "nothing to see here"},
			wantBody:      `{"error":"nothing to see here"}`,
			wantCode:      418,
		},
	}

	for _, kase := range testCases {
		errors.RegisterError(kase.err, kase.e)
	}

	for _, kase := range testCases {
		t.Run(kase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			errors.Render(context.Background(), w, kase.err)
			result := w.Result()
			assert.Equal(t, "application/error+json", result.Header.Get("Content-Type"))
			body, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			assert.JSONEq(t, kase.wantBody, string(body))
		})
	}
}

func TestErrorsRenderWithBefore(t *testing.T) {
	beforeRender := func(e E) E {
		if te, ok := e.(testE); ok {
			te.statusCode++
			return te
		}
		return e
	}
	errors := New("application/error+json", empty500{}, beforeRender, log.DefaultLogger)

	testCases := []struct {
		name          string
		err           error
		errRegistered bool
		e             E
		wantBody      string
		wantCode      int
	}{
		{
			name:          "err not registered",
			err:           fmt.Errorf("an error"),
			errRegistered: false,
			e:             DefaultE,
			wantBody:      `{}`,
			wantCode:      501,
		},
		{
			name:          "err registered",
			err:           fmt.Errorf("an error"),
			errRegistered: true,
			e:             testE{statusCode: 418, ErrorStr: "nothing to see here"},
			wantBody:      `{"error":"nothing to see here"}`,
			wantCode:      419,
		},
	}

	for _, kase := range testCases {
		errors.RegisterError(kase.err, kase.e)
	}

	for _, kase := range testCases {
		t.Run(kase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			errors.Render(context.Background(), w, kase.err)
			result := w.Result()
			assert.Equal(t, "application/error+json", result.Header.Get("Content-Type"))
			body, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			assert.JSONEq(t, kase.wantBody, string(body))
		})
	}
}

func TestErrorsRegisterReportFunc(t *testing.T) {
	errors := New(DefaultContentType, DefaultE, nil, log.DefaultLogger)

	var buf strings.Builder
	ctx := context.Background()

	reportFunc := func(ctx context.Context, err error) {
		buf.WriteString("captured ")
		buf.WriteString(err.Error())
	}

	err := fmt.Errorf("an unexpected error")

	w := httptest.NewRecorder()

	// before register the reportFunc
	errors.Render(ctx, w, err)
	assert.Equal(t, "", buf.String())

	errors.RegisterReportFunc(reportFunc)
	defer errors.RegisterReportFunc(nil)

	// after register the reportFunc
	want := "captured an unexpected error"
	errors.Render(ctx, w, err)
	assert.Equal(t, want, buf.String())
}

func TestErrorsUnRegisterErrors(t *testing.T) {
	problem := New(DefaultContentType, DefaultE, nil, log.DefaultLogger)

	problem.RegisterError(context.DeadlineExceeded, DefaultE)
	err := problem.IsKnownError(context.DeadlineExceeded)
	assert.Error(t, err, DefaultE)

	problem.UnRegisterErrors()

	err = problem.IsKnownError(context.DeadlineExceeded)
	assert.NoError(t, err)
}

func TestErrorsIsKnownError(t *testing.T) {
	errors := New(DefaultContentType, DefaultE, nil, log.DefaultLogger)

	errors.RegisterError(context.DeadlineExceeded, DefaultE)
	defer errors.UnRegisterErrors()
	err := errors.IsKnownError(context.DeadlineExceeded)
	assert.Error(t, err, DefaultE)

	err = errors.IsKnownError(fmt.Errorf("foo"))
	assert.NoError(t, err)
}
