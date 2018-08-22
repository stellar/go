package amount_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/xdr"
)

var Tests = []struct {
	S     string
	I     xdr.Int64
	valid bool
}{
	{"100.0000000", 1000000000, true},
	{"-100.0000000", -1000000000, true},
	{"100.0000001", 1000000001, true},
	{"123.0000001", 1230000001, true},
	{"123.00000001", 0, false},
	{"922337203685.4775807", 9223372036854775807, true},
	{"922337203685.4775808", 0, false},
	{"922337203686", 0, false},
	{"-922337203685.4775808", -9223372036854775808, true},
	{"-922337203685.4775809", 0, false},
	{"-922337203686", 0, false},
	{"1000000000000.0000000", 0, false},
	{"1000000000000", 0, false},
	{"-0.5000000", -5000000, true},
	{"0.5000000", 5000000, true},
	{"0.12345678", 0, false},
	// Expensive inputs:
	{strings.Repeat("1", 1000000), 0, false},
	{"1E9223372036854775807", 0, false},
	{"1e9223372036854775807", 0, false},
	{"Inf", 0, false},
}

func TestParse(t *testing.T) {
	for _, v := range Tests {
		o, err := amount.Parse(v.S)
		if !v.valid && err == nil {
			t.Errorf("expected err for input %s", v.S)
			continue
		}
		if v.valid && err != nil {
			t.Errorf("couldn't parse %s: %v", v.S, err)
			continue
		}

		if o != v.I {
			t.Errorf("%s parsed to %d, not %d", v.S, o, v.I)
		}
	}
}

func TestString(t *testing.T) {
	for _, v := range Tests {
		if !v.valid {
			continue
		}

		o := amount.String(v.I)

		if o != v.S {
			t.Errorf("%d stringified to %s, not %s", v.I, o, v.S)
		}
	}
}

func TestIntStringToAmount(t *testing.T) {
	var testCases = []struct {
		Output  string
		Input   string
		Options []amount.Option
		Valid   bool
	}{
		{"100.0000000", "1000000000", []amount.Option{}, true},
		{"-100.0000000", "-1000000000", []amount.Option{}, true},
		{"100.0000001", "1000000001", []amount.Option{}, true},
		{"123.0000001", "1230000001", []amount.Option{}, true},
		{"922337203685.4775807", "9223372036854775807", []amount.Option{}, true},
		{"922337203685.4775808", "9223372036854775808", []amount.Option{}, true},
		{"92233.7203686", "922337203686", []amount.Option{}, true},
		{"-922337203685.4775808", "-9223372036854775808", []amount.Option{}, true},
		{"-922337203685.4775809", "-9223372036854775809", []amount.Option{}, true},
		{"-92233.7203686", "-922337203686", []amount.Option{}, true},
		{"1000000000000.0000000", "10000000000000000000", []amount.Option{}, true},
		{"0.0000000", "0", []amount.Option{}, true},
		{"", "nan", []amount.Option{}, false},
		// Expensive inputs:
		{"", strings.Repeat("1", 1000000), []amount.Option{}, false},
		{"", "1E9223372036854775807", []amount.Option{}, false},
		{"", "1e9223372036854775807", []amount.Option{}, false},
		{"", "Inf", []amount.Option{}, false},
		// Expensive input but AllowOverInt64:
		{"10000000000000.0000000", "1" + strings.Repeat("0", 20), []amount.Option{amount.AllowOverInt64}, true},
		{"1" + strings.Repeat("0", 1000-7) + ".0000000", "1" + strings.Repeat("0", 1000), []amount.Option{amount.AllowOverInt64}, true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s to %s (valid = %t)", tc.Input, tc.Output, tc.Valid), func(t *testing.T) {
			o, err := amount.IntStringToAmount(tc.Input, tc.Options...)

			if !tc.Valid && err == nil {
				t.Errorf("expected err for input %s (output: %s)", tc.Input, tc.Output)
				return
			}
			if tc.Valid && err != nil {
				t.Errorf("couldn't parse %s: %v", tc.Input, err)
				return
			}

			if o != tc.Output {
				t.Errorf("%s converted to %s, not %s", tc.Input, o, tc.Output)
			}
		})
	}

}
