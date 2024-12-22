package proxy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/manager"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/conter/version"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"io"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

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

// ServeHTTP will route the HTTP request through to the desired proxy.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.IsAcmeChallenge(r) {
		// ACME is allowed on plain HTTP
		s.HandleAcmeChallenge(w, r)
		return
	}

	if r.TLS == nil {
		// always try to upgrade to HTTPS before continuing
		s.logger.Debugf("Incoming HTTP request from ip=%s redirecting to HTTPS (host=%s, url=%s)", r.RemoteAddr, r.Host, r.RequestURI)
		http.Redirect(w, r, RewriteToHTTPS(r.Host, r.RequestURI), http.StatusMovedPermanently)
		return
	}

	host := ExtractDomain(r.Host)
	route, err := s.IngressManager.Match(host)
	if err != nil {
		s.logger.Warningf("No route found for domain=%s, aborting... (ip=%s, agent=%s)", host, r.RemoteAddr, r.UserAgent())
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

// getCertificate handles the retrieval of the TLS certificate based on the SNI of the server.
func (s *Server) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hello.ServerName == "" {
		return nil, errors.New("no mappings available without a domain")
	}

	cert := s.CertificateManager.Get(hello.ServerName)
	if cert == nil {
		// No certificate found, generate a self-signed certificate
		selfSignedCert, err := generateSelfSignedCertificate(hello.ServerName)
		if err != nil {
			return nil, err
		}

		s.logger.Debugf("No certificate available, generated temporary self-signed certificate for domain=%s", hello.ServerName)
		return selfSignedCert, nil
	}

	s.logger.Tracef("Returning certificcate for domain=%s", hello.ServerName)
	return cert.X509KeyPair()
}

// generateSelfSignedCertificate generates a self-signed TLS certificate for the given domain.
func generateSelfSignedCertificate(domain string) (*tls.Certificate, error) {
	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour), // Valid for 1 day
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
		Issuer:                pkix.Name{CommonName: "conter"},
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	// Encode certificate and key to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Load the certificate and key into tls.Certificate
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tlsCert, nil
}
