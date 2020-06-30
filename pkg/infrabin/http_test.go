package infrabin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maruina/go-infrabin/internal/helpers"
)


func TestHeadersHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/headers", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-Id", "Test-Header")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HeadersHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var expected helpers.Response
	expected.Headers = &http.Header{
		"X-Request-Id": []string{"Test-Header"},
	}
	data := helpers.MarshalResponseToString(expected)

	if rr.Body.String() != data {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), data)
	}
}
