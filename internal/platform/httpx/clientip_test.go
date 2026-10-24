package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

func TestClientIP(t *testing.T) {
	proxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
	tests := []struct {
		name    string
		remote  string
		xff     []string
		trusted []netip.Prefix
		want    string
	}{
		{"peer", "203.0.113.7:51234", nil, nil, "203.0.113.7"},
		{"untrusted peer's header is ignored", "203.0.113.7:51234", []string{"198.51.100.1"}, proxies, "203.0.113.7"},
		{"no trusted proxies configured", "10.0.0.1:443", []string{"198.51.100.1"}, nil, "10.0.0.1"},
		{"IPv6 peer", "[2001:db8::1]:443", nil, nil, "2001:db8::1"},
		{"bare IPv6 peer", "2001:db8::1", nil, nil, "2001:db8::1"},
		{"bare IPv4 peer", "203.0.113.7", nil, nil, "203.0.113.7"},
		{"IPv4-mapped peer", "[::ffff:203.0.113.7]:80", nil, nil, "203.0.113.7"},
		{"zoned peer", "[fe80::1%eth0]:80", nil, nil, "fe80::1"},
		{"unix socket", "@", nil, proxies, "invalid IP"},
		{"trusted peer without a header", "10.0.0.1:443", nil, proxies, "10.0.0.1"},
		{"one proxy", "10.0.0.1:443", []string{"198.51.100.1"}, proxies, "198.51.100.1"},
		{"two proxies", "10.0.0.1:443", []string{"198.51.100.1, 10.0.0.2"}, proxies, "198.51.100.1"},
		{"spoofed hops left of the client", "10.0.0.1:443", []string{"1.1.1.1, 8.8.8.8, 198.51.100.1, 10.0.0.2"}, proxies, "198.51.100.1"},
		{"several header lines", "10.0.0.1:443", []string{"1.1.1.1", "198.51.100.1, 10.0.0.2"}, proxies, "198.51.100.1"},
		{"IPv6 hop", "[fd00::1]:443", []string{"2001:db8::7, fd00::2"}, proxies, "2001:db8::7"},
		{"hop with a port", "10.0.0.1:443", []string{"198.51.100.1:5555"}, proxies, "198.51.100.1"},
		{"bracketed IPv6 hop with a port", "10.0.0.1:443", []string{"[2001:db8::7]:5555"}, proxies, "2001:db8::7"},
		{"bracketed IPv6 hop", "10.0.0.1:443", []string{"[2001:db8::7]"}, proxies, "2001:db8::7"},
		{"garbage stops at the last proxy", "10.0.0.1:443", []string{"198.51.100.1, unknown, 10.0.0.2"}, proxies, "10.0.0.2"},
		{"empty hop", "10.0.0.1:443", []string{"198.51.100.1,"}, proxies, "10.0.0.1"},
		{"only proxies", "10.0.0.1:443", []string{"10.0.0.3, 10.0.0.2"}, proxies, "10.0.0.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remote
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := httpx.ClientIP(r, tt.trusted); got.String() != tt.want {
				t.Errorf("ClientIP = %s, want %s", got, tt.want)
			}
		})
	}
}
