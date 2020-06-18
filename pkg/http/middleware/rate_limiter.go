package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/vardius/gorouter/v4"
	"golang.org/x/time/rate"

	"github.com/vardius/go-api-boilerplate/pkg/http/request"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

type visitor struct {
	*rate.Limiter

	lastSeen time.Time
}

type rateLimiter struct {
	sync.RWMutex

	burst    int
	rate     rate.Limit
	visitors map[string]*visitor
}

// allow checks if given ip has not exceeded rate limit
func (l *rateLimiter) allow(ip string) bool {
	l.RLock()
	v, exists := l.visitors[ip]
	l.RUnlock()

	if !exists {
		v = &visitor{
			Limiter: rate.NewLimiter(l.rate, l.burst),
		}
		l.Lock()
		l.visitors[ip] = v
		l.Unlock()
	}

	v.lastSeen = time.Now()

	m.rl.Add(ip, 1)

	return v.Allow()
}

// cleanup deletes old entries
func (l *rateLimiter) cleanup(frequency time.Duration) {
	for {
		time.Sleep(frequency)

		l.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > frequency {
				delete(l.visitors, ip)

				m.rl.Delete(ip)
			}
		}
		l.Unlock()
	}
}

// RateLimit returns a new HTTP middleware that allows request per visitor (IP)
// up to rate r and permits bursts of at most b tokens.
func RateLimit(logger *log.Logger, r rate.Limit, b int, frequency time.Duration) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		if r == rate.Inf {
			return next
		}

		rl := &rateLimiter{
			rate:     r,
			burst:    b,
			visitors: make(map[string]*visitor),
		}

		go rl.cleanup(frequency)

		fn := func(w http.ResponseWriter, r *http.Request) {
			ip, err := request.IpAddress(r)
			if err != nil {
				logger.Error(r.Context(), "[HTTP] RateLimit invalid IP Address: %v\n", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if rl.allow(string(ip)) {
				next.ServeHTTP(w, r)
			}

			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		return http.HandlerFunc(fn)
	}
}
