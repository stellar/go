package httpjson

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRawObjectMarshaler(t *testing.T) {
	var in RawObject
	got, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	want := []byte("{}")
	if !bytes.Equal(got, want) {
		t.Errorf("got: %s, want: %s", string(got), string(want))
	}

	var inField struct {
		Input RawObject `json:"input"`
	}

	got, err = json.Marshal(inField)
	if err != nil {
		t.Fatal(err)
	}

	want = []byte(`{"input":{}}`)
	if !bytes.Equal(got, want) {
		t.Errorf("got: %s, want: %s", string(got), string(want))
	}
}

func TestRawObjectUnmarshaler(t *testing.T) {
	cases := []struct {
		input   []byte
		wantErr bool
	}{
		{[]byte(`{"input":{}}`), false},              // empty object
		{[]byte(`{"input":{"key":"value"}}`), false}, // object
		{[]byte(`{"input":null}`), false},            // null
		{[]byte(`{"input":[]}`), true},               // empty array
		{[]byte(`{"input":"json string"}`), true},    // string
		{[]byte(`{"input":10}`), true},               // positive number
		{[]byte(`{"input":-10}`), true},              // negative number
		{[]byte(`{"input":false}`), true},            // boolean
		{[]byte(`{"input":true}`), true},             // boolean
	}

	for _, tc := range cases {
		var out struct {
			Input RawObject `json:"input"`
		}

		err := json.Unmarshal(tc.input, &out)
		if tc.wantErr {
			if err != ErrNotJSONObject {
				t.Errorf("case %s wanted error but did not", string(tc.input))
			}
			continue
		}
		if err != nil {
			t.Errorf("case %s got error %v but shouldn't", string(tc.input), err)
		}
	}
}
