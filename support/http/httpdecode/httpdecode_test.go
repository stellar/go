package httpdecode

import (
	"bufio"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeQuery_valid(t *testing.T) {
	q := "foo=bar&list=a&list=b&enc=%2B+-%2F"
	r, _ := http.NewRequest("POST", "/?"+q, nil)

	queryDecoded := struct {
		Foo  string   `query:"foo"`
		List []string `query:"list"`
		Enc  string   `query:"enc"`
	}{}
	err := DecodeQuery(r, &queryDecoded)
	require.NoError(t, err)

	assert.Equal(t, "bar", queryDecoded.Foo)
	assert.ElementsMatch(t, []string{"a", "b"}, queryDecoded.List)
	assert.Equal(t, "+ -/", queryDecoded.Enc)
}

func TestDecodeQuery_validNone(t *testing.T) {
	r, _ := http.NewRequest("POST", "/", nil)

	queryDecoded := struct {
		Foo  string   `query:"foo"`
		List []string `query:"list"`
		Enc  string   `query:"enc"`
	}{}
	err := DecodeQuery(r, &queryDecoded)
	require.NoError(t, err)

	assert.Equal(t, "", queryDecoded.Foo)
	assert.Empty(t, queryDecoded.List)
	assert.Equal(t, "", queryDecoded.Enc)
}

// Test that DecodeQuery ignores query parameters that are invalid in the same
// way that reading out query parameters that are invalid is normally ignored
// with the built-in net/http package.
func TestDecodeQuery_invalid(t *testing.T) {
	req := `GET /?far=baf&enc=%2%B+-%2F&foo=bar HTTP/1.1

`
	r, err := http.ReadRequest(bufio.NewReader(strings.NewReader(req)))
	require.NoError(t, err)

	queryDecoded := struct {
		Far string `query:"far"`
		Enc string `query:"enc"`
		Foo string `query:"foo"`
	}{}
	err = DecodeQuery(r, &queryDecoded)
	require.NoError(t, err)

	assert.Equal(t, "baf", queryDecoded.Far)
	assert.Equal(t, "", queryDecoded.Enc)
	assert.Equal(t, "bar", queryDecoded.Foo)
}

func TestDecodeJSON_valid(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string `json:"foo"`
	}{}
	err := DecodeJSON(r, &bodyDecoded)
	require.NoError(t, err)

	assert.Equal(t, "bar", bodyDecoded.Foo)
}

func TestDecodeJSON_invalid(t *testing.T) {
	body := `{"foo:"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string `json:"foo"`
	}{}
	err := DecodeJSON(r, &bodyDecoded)
	require.EqualError(t, err, "invalid character 'b' after object key")
}

func TestDecodeForm_valid(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.Foo)
}

func TestDecodeForm_validTags(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		FooName string `form:"foo"`
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
}

func TestDecodeForm_validIgnoresUnkownKeys(t *testing.T) {
	body := `foo=bar&foz=baz`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.Foo)
}

func TestDecodeForm_validContentTypeWithOptions(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.Foo)
}

func TestDecodeForm_invalidBody(t *testing.T) {
	body := `foo=%=`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.EqualError(t, err, `invalid URL escape "%="`)
	assert.Equal(t, "", bodyDecoded.Foo)
}

func TestDecodeForm_invalidNoContentType(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.EqualError(t, err, `content type application/x-www-form-urlencoded required for form decoding: mime: no media type`)
	assert.Equal(t, "", bodyDecoded.Foo)
}

func TestDecodeForm_invalidUnrecognizedContentType(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/xwwwformurlencoded")

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.EqualError(t, err, `content type application/x-www-form-urlencoded required for form decoding: received content type "application/xwwwformurlencoded"`)
	assert.Equal(t, "", bodyDecoded.Foo)
}

func TestDecodeForm_invalidMethodType(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("GET", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		Foo string
	}{}
	err := DecodeForm(r, &bodyDecoded)
	assert.EqualError(t, err, `method POST, PUT, or PATCH required for form decoding: request has method "GET"`)
	assert.Equal(t, "", bodyDecoded.Foo)
}

func TestDecode_validJSONNoContentType(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
}

func TestDecode_validJSONWithContentType(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
}

func TestDecode_validJSONWithContentTypeOptions(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
}

func TestDecode_validForm(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
}

func TestDecode_validFormWithContentTypeOptions(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
}

func TestDecode_cannotParseContentType(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application=json")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.EqualError(t, err, "content type could not be parsed: mime: expected slash after first token")
	assert.Equal(t, "", bodyDecoded.FooName)
}

func TestDecode_invalidJSON(t *testing.T) {
	body := `{"foo""bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.EqualError(t, err, `invalid character '"' after object key`)
	assert.Equal(t, "", bodyDecoded.FooName)
}

func TestDecode_invalidForm(t *testing.T) {
	body := `foo=%=bar`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.EqualError(t, err, `invalid URL escape "%=b"`)
	assert.Equal(t, "", bodyDecoded.FooName)
}

func TestDecode_validFormAndQuery(t *testing.T) {
	body := `foo=bar`
	r, _ := http.NewRequest("POST", "/?far=boo&foo=ba2", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
		FarName string `query:"far"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
	assert.Equal(t, "boo", bodyDecoded.FarName)
}

func TestDecode_validJSONAndQuery(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/?far=boo&foo=ba2", strings.NewReader(body))

	bodyDecoded := struct {
		FooName string `json:"foo" form:"foo"`
		FarName string `query:"far"`
	}{}
	err := Decode(r, &bodyDecoded)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bodyDecoded.FooName)
	assert.Equal(t, "boo", bodyDecoded.FarName)
}
