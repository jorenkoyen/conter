package server

import "net/http"

// Handler defines the handler type that will be used to handle individual requests
type Handler func(w http.ResponseWriter, r *http.Request) error

// Middleware defines a middleware function.
type Middleware func(next http.HandlerFunc) http.HandlerFunc

// Mux defines the multiplex handler which will perform the HTTP handling.
type Mux struct {
	original    *http.ServeMux
	middlewares []Middleware
}

// NewMux creates a new multiplexer instance.
func NewMux() *Mux {
	return &Mux{
		original:    http.NewServeMux(),
		middlewares: make([]Middleware, 0),
	}
}

// Handle will register a new handler function for the specified pattern.
func (m *Mux) Handle(pattern string, handler Handler) {
	m.original.HandleFunc(pattern, m.wrap(handler))
}

// Use will append a new global middleware handler.
func (m *Mux) Use(middleware Middleware) {
	m.middlewares = append(m.middlewares, middleware)
}

// wrap will embed the error returning Handler by wrapping it in a [http.HandlerFunc].
func (m *Mux) wrap(handler Handler) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		err := handler(writer, request)
		if err != nil {
			m.error(writer, request, err)
		}
	}
}

// error will perform the print writing of the error message.
func (m *Mux) error(w http.ResponseWriter, r *http.Request, err error) {
	// TODO: write error in common format
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func (m *Mux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	chained := m.original.ServeHTTP
	for _, middleware := range m.middlewares {
		chained = middleware(chained)
	}

	// serve HTTP
	chained(writer, request)
}
