package web

import (
	"log"
	"net/http"
	"os"
)

type (
	HandlerFn    func(c Context) error
	MiddlewareFn func(next HandlerFn) HandlerFn
	HttpServer   struct {
		Server           *http.Server
		middleware       []MiddlewareFn
		routes           map[string]HandlerFn
		logger           *log.Logger
		HTTPErrorHandler func(error, Context)
	}
)

func NewHttpServer() *HttpServer {
	s := &HttpServer{
		Server:     new(http.Server),
		middleware: make([]MiddlewareFn, 0),
		routes:     make(map[string]HandlerFn),
		logger:     log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
	}
	s.Server.Handler = s
	s.HTTPErrorHandler = s.DefaultHTTPErrorHandler

	return s
}

func (h *HttpServer) Use(middleware ...MiddlewareFn) {
	h.middleware = append(h.middleware, middleware...)
}

func (h *HttpServer) Add(method, path string, handler HandlerFn) {
	h.routes[method+path] = handler
}

func (h *HttpServer) Start(address string) error {
	h.Server.Addr = address

	return h.Server.ListenAndServe()
}

// NewContext returns a Context instance.
func (h *HttpServer) NewContext(r *http.Request, w http.ResponseWriter) Context {
	return &context{
		request:  r,
		response: w,
		handler:  NotFoundHandler,
		server:   h,
	}
}

func (h *HttpServer) findHandler(method, path string) HandlerFn {
	if r, ok := h.routes[method+path]; ok {
		return r
	}

	return NotFoundHandler
}

func (h *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := h.NewContext(r, w)

	fn := h.findHandler(r.Method, getPath(r))
	fn = applyMiddleware(fn, h.middleware...)

	if err := fn(c); err != nil {
		h.HTTPErrorHandler(err, c)
	}
}

func (h *HttpServer) DefaultHTTPErrorHandler(err error, c Context) {
	he, ok := err.(*HTTPError)
	if !ok {
		he = NewHTTPErrorWithInternal(err, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if err = c.JSON(he.Code, he.Message); err != nil {
		h.logger.Println(err)
	}
}

func getPath(r *http.Request) string {
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}

	return path
}

func applyMiddleware(h HandlerFn, middleware ...MiddlewareFn) HandlerFn {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	return h
}
