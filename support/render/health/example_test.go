package health_test

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"

	"github.com/stellar/go/support/http"
	"github.com/stellar/go/support/render/health"
)

func ExampleHandler() {
	mux := http.NewAPIMux()

	mux.Get("/health", health.PassHandler{}.ServeHTTP)

	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	resp := w.Result()

	fmt.Println("Content Type:", resp.Header.Get("Content-Type"))
	fmt.Println("Status Code:", resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Body:", string(body))

	// Output:
	// Content Type: application/health+json; charset=utf-8
	// Status Code: 200
	// Body: {
	//   "status": "pass"
	// }
}
