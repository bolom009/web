package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bolom009/web"
)

// TestRateLimiter make test for rate limiter with 5/sec
func TestRateLimiter(t *testing.T) {
	var (
		mdl     = RateLimiter(5, time.Second)
		server  = web.NewHttpServer()
		handler = func(c web.Context) error {
			return c.JSON(http.StatusOK, "test")
		}
	)

	tests := []struct {
		waitTime time.Duration
		code     int
	}{
		{waitTime: 0, code: http.StatusOK},
		{waitTime: 0, code: http.StatusOK},
		{waitTime: 0, code: http.StatusOK},
		{waitTime: 0, code: http.StatusOK},
		{waitTime: 0, code: http.StatusOK},
		{waitTime: 0, code: http.StatusTooManyRequests},
		{waitTime: time.Second, code: http.StatusOK},
	}
	for _, tt := range tests {
		time.Sleep(tt.waitTime)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		ctx := server.NewContext(req, rec)

		_ = mdl(handler)(ctx)

		if rec.Code != tt.code {
			t.Errorf("RateLimiter() got = %v, want %v", rec.Code, tt.code)
		}
	}
}
