package proxy

import (
	"testing"
)

func AssertEquals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestRewriteToHTTPS(t *testing.T) {
	// simple URL
	AssertEquals(t, "https://www.example.com", RewriteToHTTPS("www.example.com", ""))
	// with query parameters
	AssertEquals(t, "https://www.example.com?q=term", RewriteToHTTPS("www.example.com", "?q=term"))
	// with paths
	AssertEquals(t, "https://www.example.com/pages/path/about.html", RewriteToHTTPS("www.example.com", "/pages/path/about.html"))
}
