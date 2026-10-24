package mail_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"net"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
)

// fakeSMTP is just enough of an SMTP server to test a client against:
// EHLO, STARTTLS, AUTH PLAIN, MAIL, RCPT, DATA, RSET and QUIT.
type fakeSMTP struct {
	ln net.Listener
	// tls, when set, is offered with STARTTLS.
	tls *tls.Config

	mu sync.Mutex
	// failData is how many DATA commands to refuse with a 451.
	failData int
	// silent servers accept connections and never greet.
	silent   bool
	accepted chan struct{}
	received []received
	logins   []string
}

type received struct {
	from, to string
	data     string
	tls      bool
}

// newFakeSMTP starts a server; withTLS makes it offer STARTTLS with a
// certificate the returned pool trusts.
func newFakeSMTP(t *testing.T, withTLS bool) (*fakeSMTP, *x509.CertPool) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, accepted: make(chan struct{}, 16)}
	var roots *x509.CertPool
	if withTLS {
		// httptest's certificate is valid for 127.0.0.1.
		hs := httptest.NewTLSServer(nil)
		f.tls = hs.TLS.Clone()
		roots = x509.NewCertPool()
		roots.AddCert(hs.Certificate())
		hs.Close()
	}
	var wg sync.WaitGroup
	t.Cleanup(func() {
		_ = ln.Close()
		wg.Wait()
	})
	wg.Go(func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			f.accepted <- struct{}{}
			wg.Go(func() { f.serve(conn) })
		}
	})
	return f, roots
}

func (f *fakeSMTP) addr() string { return f.ln.Addr().String() }

func (f *fakeSMTP) mails() []received {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]received(nil), f.received...)
}

func (f *fakeSMTP) serve(conn net.Conn) {
	defer conn.Close()
	f.mu.Lock()
	silent := f.silent
	f.mu.Unlock()
	if silent {
		// Hold the connection until the client gives up.
		_, _ = conn.Read(make([]byte, 1))
		return
	}

	tp := textproto.NewConn(conn)
	secure := false
	var from, to string
	reply := func(lines ...string) {
		for _, l := range lines {
			if err := tp.PrintfLine("%s", l); err != nil {
				return
			}
		}
	}
	reply("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO", "HELO":
			lines := []string{"250-fake"}
			if f.tls != nil && !secure {
				lines = append(lines, "250-STARTTLS")
			}
			reply(append(lines, "250 AUTH PLAIN")...)
		case "STARTTLS":
			reply("220 go ahead")
			tlsConn := tls.Server(conn, f.tls)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn, tp, secure = tlsConn, textproto.NewConn(tlsConn), true
		case "AUTH":
			_, creds, _ := strings.Cut(arg, " ")
			decoded, err := base64.StdEncoding.DecodeString(creds)
			if err != nil {
				reply("501 bad credentials")
				continue
			}
			f.mu.Lock()
			f.logins = append(f.logins, string(decoded))
			f.mu.Unlock()
			reply("235 ok")
		case "MAIL":
			from = arg
			reply("250 ok")
		case "RCPT":
			to = arg
			reply("250 ok")
		case "DATA":
			f.mu.Lock()
			refuse := f.failData > 0
			if refuse {
				f.failData--
			}
			f.mu.Unlock()
			if refuse {
				reply("451 try again later")
				continue
			}
			reply("354 go ahead")
			lines, err := tp.ReadDotLines()
			if err != nil {
				return
			}
			f.mu.Lock()
			f.received = append(f.received, received{from: from, to: to, data: strings.Join(lines, "\r\n"), tls: secure})
			f.mu.Unlock()
			reply("250 queued")
		case "RSET", "NOOP":
			reply("250 ok")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("502 unknown command")
		}
	}
}
