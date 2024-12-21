package proxy

import "strings"

// RewriteToHTTPS will rewrite any incoming URL to the HTTPS scheme.
func RewriteToHTTPS(host, uri string) string {
	return "https://" + host + uri
}

// ExtractDomain will extract the domain name from the host parameter.
// It will drop the port number if specified.
func ExtractDomain(host string) string {
	idx := strings.Index(host, ":")
	if idx != -1 {
		return host[:idx]
	} else {
		return host
	}
}
