package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type (
	Context interface {
		Request() *http.Request
		Response() http.ResponseWriter
		BindBody(i interface{}) error
		Handler() HandlerFn
		RealIP() string
		JSON(code int, i interface{}) error
		Error(err error)
	}
	context struct {
		request  *http.Request
		response http.ResponseWriter
		path     string
		handler  HandlerFn
		lock     sync.RWMutex
		server   *HttpServer
	}
)

func (c *context) Request() *http.Request {
	return c.request
}

func (c *context) Response() http.ResponseWriter {
	return c.response
}

func (c *context) BindBody(i interface{}) error {
	ctype := c.Request().Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ctype, "application/json"):
		err := json.NewDecoder(c.Request().Body).Decode(i)
		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			return NewHTTPErrorWithInternal(err, http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset))
		} else if se, ok := err.(*json.SyntaxError); ok {
			return NewHTTPErrorWithInternal(err, http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error()))
		}
	}

	return nil
}

func (c *context) Handler() HandlerFn {
	return c.handler
}

func (c *context) RealIP() string {
	IPAddress := c.Request().Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = c.Request().Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = c.Request().RemoteAddr
	}

	return IPAddress
}

func (c *context) JSON(code int, v interface{}) (err error) {
	header := c.Response().Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}

	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.Response().WriteHeader(code)
	if _, err = c.Response().Write(b); err != nil {
		return err
	}

	return nil
}

func (c *context) Error(err error) {
	c.server.HTTPErrorHandler(err, c)
}
