package httpx

import (
	"container/list"
	"fmt"
	"math"
	"net/http"
	"net/netip"
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
	l := NewLimiter[netip.Addr](cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := l.Allow(clientKey(ClientIP(r, trusted)), time.Now()); err != nil {
				WriteError(w, r, err)
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

// TooManyRequestsError means a client has used up its rate limit. WriteError
// answers it with 429 and a Retry-After of RetryAfter, rounded up to whole
// seconds.
type TooManyRequestsError struct {
	RetryAfter time.Duration
}

func (e *TooManyRequestsError) Error() string {
	return fmt.Sprintf("too many requests; retry after %s", e.RetryAfter)
}

// retryAfterSeconds is what Retry-After says: at least one second, since 0
// would invite an immediate retry that fails again.
func (e *TooManyRequestsError) retryAfterSeconds() int {
	return max(1, int(math.Ceil(e.RetryAfter.Seconds())))
}

// Limiter keeps a token bucket per key, for limits that aren't per client
// IP, such as login links per email address. It remembers only
// cfg.Visitors keys, forgetting the one used least recently, so memory stays
// bounded whatever keys arrive. It is safe for concurrent use.
type Limiter[K comparable] struct {
	cfg RateLimitConfig

	mu      sync.Mutex
	clients map[K]*list.Element // of *client[K], front most recent
	lru     *list.List
}

type client[K comparable] struct {
	key    K
	bucket *rate.Limiter
}

// NewLimiter returns a Limiter whose buckets cfg sizes.
func NewLimiter[K comparable](cfg RateLimitConfig) *Limiter[K] {
	return &Limiter[K]{cfg: cfg.withDefaults(), clients: map[K]*list.Element{}, lru: list.New()}
}

// Allow takes a token from key's bucket at now. When the bucket is empty it
// returns a *TooManyRequestsError saying how long until a token is there.
func (l *Limiter[K]) Allow(key K, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.clients[key]
	if ok {
		l.lru.MoveToFront(e)
	} else {
		if l.lru.Len() >= l.cfg.Visitors {
			oldest := l.lru.Back()
			delete(l.clients, l.lru.Remove(oldest).(*client[K]).key)
		}
		e = l.lru.PushFront(&client[K]{key: key, bucket: rate.NewLimiter(rate.Limit(l.cfg.Rate), l.cfg.Burst)})
		l.clients[key] = e
	}
	bucket := e.Value.(*client[K]).bucket
	if bucket.AllowN(now, 1) {
		return nil
	}
	return &TooManyRequestsError{RetryAfter: time.Duration((1 - bucket.TokensAt(now)) / l.cfg.Rate * float64(time.Second))}
}
