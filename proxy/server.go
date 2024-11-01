package proxy

import (
	"context"
	"fmt"
	"github.com/jorenkoyen/conter/manager"
	"github.com/jorenkoyen/conter/model"
	"github.com/jorenkoyen/conter/version"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	AddressHTTP  = "0.0.0.0:80"
	AddressHTTPS = "0.0.0.0:443"
)

type Server struct {
	logger         *logger.Logger
	IngressManager *manager.IngressManager
	// TODO: challenge manager
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
	route, err := s.IngressManager.Match(r.Host)
	if err != nil {
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
	s.logger.Tracef("Routing through request to endpoint=%s (service=%s, method=%s, path=%s)", route.Endpoint, route.Service, r.Method, r.URL.Path)
	proxy.ServeHTTP(w, r)
}

func (s *Server) createProxyTarget(route *model.IngressRoute) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(fmt.Sprintf("http://%s", route.Endpoint))
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
		s.logger.Errorf("Failed to route request to service=%s: %v", route.Service, err)
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	return proxy, nil
}

// ListenForHTTP will start listening for incoming HTTP request that require to be proxied through.
func (s *Server) ListenForHTTP(ctx context.Context) error {
	server := &http.Server{Addr: AddressHTTP, Handler: s}

	go func() {
		<-ctx.Done()
		s.logger.Trace("Gracefully shutting down HTTP proxy")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	// create HTTP server
	s.logger.Infof("Starting HTTP proxy on address=%s", AddressHTTP)
	return server.ListenAndServe()
}
