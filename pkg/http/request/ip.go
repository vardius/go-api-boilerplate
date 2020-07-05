package request

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// IpAddress returns client ip address from request
// Will check X-Real-IP and X-Forwarded-For header.
// Unless you have a trusted reverse proxy, you shouldn't use this function, the client can set headers to any arbitrary value it wants
func IpAddress(r *http.Request) (net.IP, error) {
	addr := r.RemoteAddr
	if xReal := r.Header.Get("X-Real-Ip"); xReal != "" {
		addr = xReal
	} else if xForwarded := r.Header.Get("X-Forwarded-For"); xForwarded != "" {
		addr = xForwarded
	}

	ip := addr
	if strings.Contains(addr, ":") {
		var err error
		ip, _, err = net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("addr: %q is not ip:port %w", addr, err)
		}
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return nil, fmt.Errorf("ip: %q is not a valid IP address", ip)
	}

	return userIP, nil
}
