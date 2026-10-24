package httpx

import (
	"container/list"
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig sets RateLimit's token buckets. Services nest it with
// envPrefix:"RATE_LIMIT_". Zero fields take the defaults shown.
type RateLimitConfig struct {
	Rate     float64 `env:"RATE" envDefault:"10" desc:"Requests per second each client may make, on average."`
	Burst    int     `env:"BURST" envDefault:"20" desc:"Requests a client may make at once before the rate applies."`
	Visitors int     `env:"VISITORS" envDefault:"10000" desc:"Clients remembered at once; the one seen least recently is forgotten first."`
}

func (c RateLimitConfig) withDefaults() RateLimitConfig {
	if c.Rate <= 0 {
		c.Rate = 10
	}
	if c.Burst <= 0 {
		c.Burst = 20
	}
	if c.Visitors <= 0 {
		c.Visitors = 10000
	}
	return c
}

// RateLimit gives each client, by ClientIP with trusted as the proxies, a
// token bucket, and answers a client that has run out with a 429 problem and
// Retry-After. An IPv6 client is limited by its /64, the block one host is
// usually given, since it can pick any address within it. Only cfg.Visitors
// clients are remembered, so memory stays bounded whatever addresses
// arrive.
func RateLimit(cfg RateLimitConfig, trusted []netip.Prefix) Middleware {
	l := &limiter{cfg: cfg.withDefaults(), clients: map[netip.Addr]*list.Element{}, lru: list.New()}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wait, ok := l.allow(clientKey(ClientIP(r, trusted)), time.Now())
			if !ok {
				secs := max(1, int(math.Ceil(wait.Seconds())))
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				writeProblem(w, r, http.StatusTooManyRequests, fmt.Sprintf("Too many requests; try again in %s.", time.Duration(secs)*time.Second), nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientKey(a netip.Addr) netip.Addr {
	if a.Is6() {
		// A /64 of a valid address always exists.
		p, _ := a.Prefix(64)
		return p.Addr()
	}
	return a
}

type limiter struct {
	cfg RateLimitConfig

	mu      sync.Mutex
	clients map[netip.Addr]*list.Element // of *client, front most recent
	lru     *list.List
}

type client struct {
	key    netip.Addr
	bucket *rate.Limiter
}

// allow takes a token from key's bucket, or says how long until one is
// there.
func (l *limiter) allow(key netip.Addr, now time.Time) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.clients[key]
	if ok {
		l.lru.MoveToFront(e)
	} else {
		if l.lru.Len() >= l.cfg.Visitors {
			oldest := l.lru.Back()
			delete(l.clients, l.lru.Remove(oldest).(*client).key)
		}
		e = l.lru.PushFront(&client{key: key, bucket: rate.NewLimiter(rate.Limit(l.cfg.Rate), l.cfg.Burst)})
		l.clients[key] = e
	}
	bucket := e.Value.(*client).bucket
	if bucket.AllowN(now, 1) {
		return 0, true
	}
	return time.Duration((1 - bucket.TokensAt(now)) / l.cfg.Rate * float64(time.Second)), false
}
