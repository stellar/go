package errors

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorsRender tests the render cases
func TestRender(t *testing.T) {
	defer UnRegisterErrors()

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
		RegisterError(kase.err, kase.e)
	}

	for _, kase := range testCases {
		t.Run(kase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Render(context.Background(), w, kase.err)
			result := w.Result()
			assert.Equal(t, "application/json; charset=utf8", result.Header.Get("Content-Type"))
			body, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			assert.JSONEq(t, kase.wantBody, string(body))
		})
	}
}

func TestRegisterReportFunc(t *testing.T) {
	var buf strings.Builder
	ctx := context.Background()

	reportFunc := func(ctx context.Context, err error) {
		buf.WriteString("captured ")
		buf.WriteString(err.Error())
	}

	err := fmt.Errorf("an unexpected error")

	w := httptest.NewRecorder()

	// before register the reportFunc
	Render(ctx, w, err)
	assert.Equal(t, "", buf.String())

	RegisterReportFunc(reportFunc)
	defer RegisterReportFunc(nil)

	// after register the reportFunc
	want := "captured an unexpected error"
	Render(ctx, w, err)
	assert.Equal(t, want, buf.String())
}

func TestUnRegisterErrors(t *testing.T) {
	RegisterError(context.DeadlineExceeded, DefaultE)
	err := IsKnownError(context.DeadlineExceeded)
	assert.Error(t, err, DefaultE)

	UnRegisterErrors()

	err = IsKnownError(context.DeadlineExceeded)
	assert.NoError(t, err)
}

func TestIsKnownError(t *testing.T) {
	RegisterError(context.DeadlineExceeded, DefaultE)
	defer UnRegisterErrors()
	err := IsKnownError(context.DeadlineExceeded)
	assert.Error(t, err, DefaultE)

	err = IsKnownError(fmt.Errorf("foo"))
	assert.NoError(t, err)
}
