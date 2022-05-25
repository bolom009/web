package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/bolom009/web"
)

var ErrRateLimitExceeded = web.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")

type (
	RateLimiterStore struct {
		mutex sync.Mutex

		visitors map[string]*Visitor
		rate     int

		lastCleanup time.Time
		expiresIn   time.Duration
	}
	Visitor struct {
		// usedRate the number of used calls
		usedRate int
		// lastSeen is the time for refresh rate calls
		lastSeen time.Time
	}
)

func NewVisitor(rate int) *Visitor {
	return &Visitor{
		usedRate: rate,
		lastSeen: time.Now(),
	}
}

func (v *Visitor) Reserve() bool {
	if v.usedRate <= 0 {
		return false
	}

	v.usedRate--
	return true
}

func NewRateLimiterStore(limit int, expires time.Duration) *RateLimiterStore {
	return &RateLimiterStore{
		visitors:    make(map[string]*Visitor),
		rate:        limit,
		lastCleanup: time.Now(),
		expiresIn:   expires,
	}
}

func (s *RateLimiterStore) AllowVisitor(identifier string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if time.Now().Sub(s.lastCleanup) > s.expiresIn {
		s.cleanupVisitors()
	}

	visitor, exists := s.visitors[identifier]
	if !exists {
		visitor = NewVisitor(s.rate)
		s.visitors[identifier] = visitor
	}

	allowReserve := visitor.Reserve()
	if allowReserve {
		visitor.lastSeen = time.Now()
	}

	return allowReserve
}

func (s *RateLimiterStore) cleanupVisitors() {
	for id, visitor := range s.visitors {
		if time.Now().Sub(visitor.lastSeen) > s.expiresIn {
			delete(s.visitors, id)
		}
	}

	s.lastCleanup = time.Now()
}

func RateLimiter(limit int, expires time.Duration) web.MiddlewareFn {
	cfg := NewRateLimiterStore(limit, expires)

	return func(next web.HandlerFn) web.HandlerFn {
		return func(c web.Context) error {
			if allow := cfg.AllowVisitor(c.RealIP()); !allow {
				c.Error(ErrRateLimitExceeded)
				return nil
			}

			return next(c)
		}
	}
}
