package hal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

type handler struct {
	fv      reflect.Value
	inType  reflect.Type
	inValue []byte
}

// Handler returns an HTTP Handler for function fn.
// If fn returns a non-nil error, the handler will use problem.Render.
func Handler(fn, param interface{}) (http.Handler, error) {
	fv := reflect.ValueOf(fn)
	inType, err := funcParamType(fv)
	if err != nil {
		return nil, errors.Wrap(err, "parsing function prototype")
	}

	inValue, err := json.Marshal(param)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling function input value")
	}

	return &handler{fv, inType, inValue}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	res, err := h.executeFunc(ctx)
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	Render(w, res)
}

func (h *handler) executeFunc(ctx context.Context) (interface{}, error) {
	var a []reflect.Value
	a = append(a, reflect.ValueOf(ctx))
	if h.inType != nil {
		inPtr := reflect.New(h.inType)
		err := json.Unmarshal(h.inValue, inPtr.Interface())
		if err != nil {
			return nil, err
		}
		a = append(a, inPtr.Elem())
	}

	rv := h.fv.Call(a)

	return rv[0].Interface(), rv[1].Interface().(error)
}

func ExecuteFunc(ctx context.Context, fn, param interface{}) (interface{}, error) {
	h, err := Handler(fn, param)
	if err != nil {
		return nil, err
	}

	return h.(*handler).executeFunc(ctx)
}

func funcParamType(fv reflect.Value) (reflect.Type, error) {
	ft := fv.Type()
	if ft.Kind() != reflect.Func || ft.IsVariadic() || ft.NumIn() > 2 {
		return nil, fmt.Errorf("%s must be nonvariadic func and has at most one parameter other than context", ft.String())
	}

	var paramType reflect.Type
	if ft.NumIn() == 2 {
		// the first param is context
		paramType = ft.In(1)
	}

	if ft.NumOut() != 2 || !ft.Out(1).Implements(errorType) {
		return nil, fmt.Errorf("%s must have two return values, and the second return value must be an error", ft.String())
	}

	return paramType, nil
}
