package httpx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

type limited struct {
	t *testing.T
	h http.Handler
}

func newLimited(t *testing.T, cfg httpx.RateLimitConfig, trusted ...netip.Prefix) limited {
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	return limited{t, httpx.RateLimit(cfg, trusted)(ok)}
}

// from sends one request from remote, forwarded for xff when set, and
// returns its status.
func (l limited) from(remote, xff string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/v1/login-links", nil)
	r.RemoteAddr = remote
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return serve(l.h, r)
}

func (l limited) expect(remote, xff string, want ...int) {
	l.t.Helper()
	for i, w := range want {
		if got := l.from(remote, xff).Code; got != w {
			l.t.Errorf("request %d from %s (%s) = %d, want %d", i+1, remote, xff, got, w)
		}
	}
}

func TestRateLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{Rate: 1, Burst: 2})
		l.expect("203.0.113.7:1", "", 200, 200)

		rec := l.from("203.0.113.7:2", "")
		if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "1" {
			t.Fatalf("over the limit = %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
		}
		if p := readProblem(t, rec); p["detail"] != "Too many requests; try again in 1s." {
			t.Errorf("problem = %v", p)
		}

		l.expect("198.51.100.1:1", "", 200)
		time.Sleep(time.Second)
		l.expect("203.0.113.7:3", "", 200, 429)
	})
}

func TestRateLimitRetryAfterRoundsUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{Rate: 0.4, Burst: 1})
		l.expect("203.0.113.7:1", "", 200)
		if got := l.from("203.0.113.7:1", "").Header().Get("Retry-After"); got != "3" {
			t.Errorf("Retry-After = %q, want 3 (2.5s rounded up)", got)
		}
	})
}

func TestRateLimitKeysOnTheClientBehindProxies(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{Rate: 1, Burst: 1}, netip.MustParsePrefix("10.0.0.0/8"))

		// A client that isn't a proxy can't pick its key with the header.
		l.expect("203.0.113.7:1", "1.1.1.1", 200)
		l.expect("203.0.113.7:1", "2.2.2.2", 429)

		// Behind the proxy, each client has its own bucket.
		l.expect("10.0.0.1:1", "198.51.100.1", 200, 429)
		l.expect("10.0.0.1:1", "198.51.100.2", 200)
		l.expect("10.0.0.1:1", "6.6.6.6, 198.51.100.2", 429)
	})
}

func TestRateLimitGroupsIPv6ByPrefix(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{Rate: 1, Burst: 1})
		l.expect("[2001:db8:1:2::1]:1", "", 200)
		l.expect("[2001:db8:1:2:ffff::9]:1", "", 429)
		l.expect("[2001:db8:1:3::1]:1", "", 200)
	})
}

func TestRateLimitForgetsTheLeastRecentClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{Rate: 1, Burst: 1, Visitors: 2})
		l.expect("192.0.2.1:1", "", 200)
		l.expect("192.0.2.2:1", "", 200)
		l.expect("192.0.2.1:1", "", 429) // now the most recent
		l.expect("192.0.2.3:1", "", 200) // forgets .2
		l.expect("192.0.2.1:1", "", 429)
		l.expect("192.0.2.2:1", "", 200)
	})
}

func TestRateLimitDefaults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{})
		for range 20 {
			l.expect("192.0.2.1:1", "", 200)
		}
		l.expect("192.0.2.1:1", "", 429)
		time.Sleep(100 * time.Millisecond)
		l.expect("192.0.2.1:1", "", 200)
	})
}

func TestLimiterKeepsABucketPerKey(t *testing.T) {
	l := httpx.NewLimiter[string](httpx.RateLimitConfig{Rate: 0.5, Burst: 2, Visitors: 2})
	now := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

	for i := range 2 {
		if err := l.Allow("a@example.com", now); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	err := l.Allow("a@example.com", now)
	tmr, ok := errors.AsType[*httpx.TooManyRequestsError](err)
	if !ok || tmr.RetryAfter != 2*time.Second {
		t.Fatalf("over the limit: err = %v, want a TooManyRequestsError retrying after 2s", err)
	}
	if err := l.Allow("b@example.com", now); err != nil {
		t.Errorf("another key shares the bucket: %v", err)
	}
	if err := l.Allow("a@example.com", now.Add(2*time.Second)); err != nil {
		t.Errorf("a token didn't come back after 2s: %v", err)
	}

	// A third key makes the limiter forget the least recent one, b.
	if err := l.Allow("c@example.com", now); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if err := l.Allow("b@example.com", now); err != nil {
			t.Errorf("b request %d after being forgotten: %v", i+1, err)
		}
	}
}

func TestRateLimitAnswersWithTheProblemTable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimited(t, httpx.RateLimitConfig{Rate: 1, Burst: 1})
		l.expect("203.0.113.7:1", "", 200)
		rec := l.from("203.0.113.7:1", "")
		if p := readProblem(t, rec); p["status"] != float64(http.StatusTooManyRequests) || p["instance"] != "/v1/login-links" {
			t.Errorf("problem = %v", p)
		}
	})
}
