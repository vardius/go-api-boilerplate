package middleware

import (
	"expvar"
	"fmt"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	"net/http"
	"sync"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/http/request"
	"github.com/vardius/gorouter/v4"
	"golang.org/x/time/rate"
)

var rateLimits *expvar.Map

type visitor struct {
	*rate.Limiter

	lastSeen time.Time
}

type rateLimiter struct {
	sync.Mutex

	burst    int
	rate     rate.Limit
	visitors map[string]*visitor
}

// allow checks if given ip has not exceeded rate limit
func (l *rateLimiter) allow(ip string) bool {
	l.Lock()
	defer l.Unlock()

	v, exists := l.visitors[ip]
	if !exists {
		v = &visitor{
			Limiter: rate.NewLimiter(l.rate, l.burst),
		}
		l.visitors[ip] = v
	}

	v.lastSeen = time.Now()

	if rateLimits != nil {
		rateLimits.Add(ip, 1)
	}

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

				if rateLimits != nil {
					rateLimits.Delete(ip)
				}
			}
		}
		l.Unlock()
	}
}

// RateLimit returns a new HTTP middleware that allows request per visitor (IP)
// up to rate r and permits bursts of at most b tokens.
// Please add before metrics if used together
func RateLimit(r rate.Limit, b int, frequency time.Duration) gorouter.MiddlewareFunc {
	// If metrics middleware was used
	if hasMetrics {
		rateLimits = expvar.NewMap("rateLimits")
	}

	var rl *rateLimiter
	if r != rate.Inf {
		rl = &rateLimiter{
			rate:     r,
			burst:    b,
			visitors: make(map[string]*visitor),
		}

		go rl.cleanup(frequency)
	}

	return func(next http.Handler) http.Handler {
		if r == rate.Inf {
			return next
		}

		fn := func(w http.ResponseWriter, r *http.Request) {
			ip, err := request.IpAddress(r)
			if err != nil {
				logger.Error(r.Context(), fmt.Sprintf("[HTTP] RateLimit invalid IP Address: %v", err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if !rl.allow(string(ip)) {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
