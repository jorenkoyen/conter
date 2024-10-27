package server

import (
	"net/http"
	"strings"
)

// IsJson will check if the Content-Type of the request is application/json
func IsJson(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
}
