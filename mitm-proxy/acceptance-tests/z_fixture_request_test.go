package acceptance_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

func NewPOSTRequest(target, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	// Permit both direct handler calls and http.Client.Do.
	request.RequestURI = ""
	return request
}
