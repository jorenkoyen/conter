package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/manager"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/conter/version"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	defaultLog "log"
)

type Server struct {
	logger             *logger.Logger
	IngressManager     *manager.IngressManager
	CertificateManager *manager.CertificateManager
}

func NewServer() *Server {
	return &Server{
		logger: log.WithName("proxy"),
	}
}

// SetLogLevel overrides the log level for the reverse proxy logger.
func (s *Server) SetLogLevel(l logger.Level) {
	s.logger.SetLogLevel(l)
}

// ServeHTTP will route the HTTP request through to the desired proxy.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.IsAcmeChallenge(r) {
		s.HandleAcmeChallenge(w, r)
		return
	}

	route, err := s.IngressManager.Match(r.Host)
	if err != nil {
		s.logger.Warningf("No route found for domain=%s, aborting... (ip=%s, agent=%s)", r.Host, r.RemoteAddr, r.UserAgent())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// proxy through request to endpoint
	proxy, err := s.createProxyTarget(route)
	if err != nil {
		s.logger.Errorf("Failed to create proxy target: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// proxy request
	s.logger.Tracef("Routing through request to endpoint=%s (service=%s, method=%s, path=%s)", route.TargetEndpoint, route.TargetService, r.Method, r.URL.Path)
	proxy.ServeHTTP(w, r)
}

func (s *Server) createProxyTarget(ingress *types.Ingress) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(fmt.Sprintf("http://%s", ingress.TargetEndpoint))
	if err != nil {
		return nil, err
	}

	// Create a reverse proxy pointing to the target URL
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = func(r *http.Response) error {
		// overwrite Server header
		r.Header.Set("Server", fmt.Sprintf("conter/%s", version.Version))
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Errorf("Failed to route request to service=%s: %v", ingress.TargetService, err)
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	return proxy, nil
}

// ListenForHTTP will start listening for incoming HTTP request that require to be proxied through.
func (s *Server) ListenForHTTP(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:     addr,
		Handler:  s,
		ErrorLog: defaultLog.New(io.Discard, "", 0),
	}

	go func() {
		<-ctx.Done()
		s.logger.Trace("Gracefully shutting down HTTP proxy")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	// create HTTP server
	s.logger.Debugf("Starting HTTP proxy on address=%s", addr)
	return server.ListenAndServe()
}

// ListenForHTTPS will start listening for incoming HTTPS requests that required to be proxied through.
func (s *Server) ListenForHTTPS(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: s,
		TLSConfig: &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: s.getCertificate,
		},
		ErrorLog: defaultLog.New(io.Discard, "", 0),
	}

	go func() {
		<-ctx.Done()
		s.logger.Trace("Gracefully shutting down HTTPS proxy")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	// create HTTPS server
	s.logger.Debugf("Starting HTTPS proxy on address=%s", addr)
	return server.ListenAndServeTLS("", "")
}

// getCertificate handles the retrieval of the TLS certificate based on the SNI of the server.
func (s *Server) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert := s.CertificateManager.Get(hello.ServerName)
	if cert == nil {
		return nil, errors.New("no certificate found")
	}

	// TODO: check if certificate is valid (not expired)

	s.logger.Tracef("Returning certificcate for domain=%s", hello.ServerName)
	return cert.X509KeyPair()
}
