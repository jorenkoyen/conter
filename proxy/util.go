package proxy

import "net/url"

// RewriteToHTTPS will rewrite any incoming URL to the HTTPS scheme.
func RewriteToHTTPS(inbound *url.URL) string {
	inbound.Scheme = "https"
	return inbound.String()
}
