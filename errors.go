package web

import (
	"fmt"
	"net/http"
)

func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}

	return he
}

func NewHTTPErrorWithInternal(internalError error, code int, message ...interface{}) *HTTPError {
	e := NewHTTPError(code, message...)
	e.InternalError = internalError
	return e
}

var NotFoundHandler = func(c Context) error {
	return NewHTTPError(http.StatusNotFound)
}

type HTTPError struct {
	Code          int         `json:"-"`
	Message       interface{} `json:"message"`
	InternalError error       `json:"-"`
}

func (h *HTTPError) Error() string {
	if h.InternalError == nil {
		return fmt.Sprintf("code=%d, message=%v", h.Code, h.Message)
	}

	return fmt.Sprintf("code=%d, message=%v internal=%v", h.Code, h.Message, h.InternalError)
}
