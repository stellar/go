package httpdecode

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
