package proxy

import (
	"net/http"
	"strings"
)

const AcmePrefix = "/.well-known/acme-challenge/"

// IsAcmeChallenge will return true when the request contains the expected path for an ACME challenge.
func (s *Server) IsAcmeChallenge(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, AcmePrefix)
}

// HandleAcmeChallenge will handle an incoming ACME request.
func (s *Server) HandleAcmeChallenge(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, AcmePrefix)
	host := r.Host

	// drop any port mappings from host
	idx := strings.Index(host, ":")
	if idx != -1 {
		host = host[:idx]
	}

	s.logger.Infof("Handling incoming ACME request for host=%s (token=%s)", host, token)
	auth, err := s.CertificateManager.Authorize(host, token)
	if err != nil {
		s.logger.Errorf("Invalid challenge token for host=%s: %v", host, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(auth))
}
