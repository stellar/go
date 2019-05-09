package hal

import (
	"context"
	"reflect"
	"testing"
)

func TestFuncParamTypeError(t *testing.T) {
	cases := []interface{}{
		0,                                        // not a function
		"a string",                               // not a function
		func() (int, error) { return 0, nil },    // no inputs
		func(int) (int, error) { return 0, nil }, // first input is not context
		func(context.Context) {},                 // not return values
		func(context.Context, int, int) (int, error) { return 0, nil }, // too many inputs
		func(context.Context, int) (int, int) { return 0, 0 },          // second return value is not an error
		func() (int, int, error) { return 0, 0, nil },                  // too many return values
	}

	for _, tc := range cases {
		_, err := funcParamType(reflect.ValueOf(tc))
		if err == nil {
			t.Errorf("funcParamType(%T) wants error", tc)
		}
	}
}
