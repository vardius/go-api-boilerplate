package httpx

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// ClientIP returns the address of the client that sent r. That is the peer,
// r.RemoteAddr, unless the peer is a trusted proxy: then it is the
// right-most X-Forwarded-For hop that isn't one, since everything left of
// it was written by the client and can't be believed. A hop that doesn't
// parse ends the walk at the proxy that passed it on. IPv4-mapped IPv6
// addresses are unmapped and zones dropped. It returns the zero Addr when
// RemoteAddr isn't an IP address, as on a Unix socket.
func ClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	addr := parseHop(r.RemoteAddr)
	if !addr.IsValid() || !isTrusted(addr, trusted) {
		return addr
	}
	headers := r.Header.Values("X-Forwarded-For")
	for _, header := range slices.Backward(headers) {
		for _, raw := range slices.Backward(strings.Split(header, ",")) {
			hop := parseHop(strings.TrimSpace(raw))
			if !hop.IsValid() {
				return addr
			}
			if !isTrusted(hop, trusted) {
				return hop
			}
			addr = hop
		}
	}
	return addr
}

// parseHop accepts an address with or without a port, IPv6 with or
// without brackets.
func parseHop(s string) netip.Addr {
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap().WithZone("")
	}
	if a, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")); err == nil {
		return a.Unmap().WithZone("")
	}
	return netip.Addr{}
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	return slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(a) })
}
