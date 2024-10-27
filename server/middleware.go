package server

import (
	"net/http"
	"time"
)

// LoggerMiddleware will output the HTTP activity.
func (s *Server) LoggerMiddleware() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next(w, r)

			end := time.Since(start)
			path := r.URL.Path
			method := r.Method
			s.logger.Tracef("Request completed [ method=%s path=%s dur=%dms, ip=%s ]", method, path, end.Milliseconds(), r.RemoteAddr)
		}
	}
}
