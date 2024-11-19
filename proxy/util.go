package proxy

// RewriteToHTTPS will rewrite any incoming URL to the HTTPS scheme.
func RewriteToHTTPS(host, uri string) string {
	return "https://" + host + uri
}
