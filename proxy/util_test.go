package proxy

import (
	"net/url"
	"testing"
)

func AssertEquals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func ParseUrl(t *testing.T, input string) *url.URL {
	t.Helper()
	u, err := url.Parse(input)
	if err != nil {
		t.Errorf("Failed to parse url=%s: %v", input, err)
		t.FailNow()
	}
	return u
}

func TestRewriteToHTTPS(t *testing.T) {
	{
		// simple URL
		inbound := ParseUrl(t, "http://www.example.com")
		AssertEquals(t, "https://www.example.com", RewriteToHTTPS(inbound))
	}

	{
		// with query parameters
		inbound := ParseUrl(t, "http://www.example.com?q=term")
		AssertEquals(t, "https://www.example.com?q=term", RewriteToHTTPS(inbound))
	}

	{
		// with paths
		inbound := ParseUrl(t, "http://www.example.com/pages/path/about.html")
		AssertEquals(t, "https://www.example.com/pages/path/about.html", RewriteToHTTPS(inbound))
	}

	{
		// with segment
		inbound := ParseUrl(t, "http://www.example.com/about.html#contact")
		AssertEquals(t, "https://www.example.com/about.html#contact", RewriteToHTTPS(inbound))
	}
}
