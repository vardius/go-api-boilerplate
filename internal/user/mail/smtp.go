package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

// SMTPConfig says how to reach the mail server. The service nests it with
// envPrefix:"SMTP_".
type SMTPConfig struct {
	Addr          string        `env:"ADDR,required" desc:"SMTP server, host:port, such as smtp.example.com:587."`
	Username      string        `env:"USERNAME" desc:"SMTP user name; empty sends without logging in."`
	Password      config.Secret `env:"PASSWORD" desc:"SMTP password."`
	From          string        `env:"FROM,required" desc:"Sender of every email: an address, or Name <address>."`
	AllowInsecure bool          `env:"ALLOW_INSECURE" envDefault:"false" desc:"Send in plain text when the server doesn't offer STARTTLS. Only for a local mail catcher such as Mailpit."`
}

// sendTimeout bounds one Send when its context sets no earlier deadline, so
// a server that stops answering can't hold a delivery forever.
const sendTimeout = 30 * time.Second

// SMTP is a Mailer that sends through an SMTP server, upgrading the
// connection with STARTTLS. A server that doesn't offer STARTTLS is refused
// unless AllowInsecure is set, and credentials are never sent in plain text
// to anything but localhost.
type SMTP struct {
	addr          string
	host          string
	from          *mail.Address
	username      string
	password      config.Secret
	allowInsecure bool

	// TLS configures STARTTLS. Nil verifies the server's certificate
	// against the system roots; tests set their own roots.
	TLS *tls.Config
	// Now stamps the Date header. It defaults to time.Now.
	Now func() time.Time
}

var _ Mailer = (*SMTP)(nil)

// NewSMTP returns an SMTP Mailer for cfg, checking its address and sender.
func NewSMTP(cfg SMTPConfig) (*SMTP, error) {
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("mail: SMTP_ADDR %q: %w", cfg.Addr, err)
	}
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("mail: SMTP_FROM %q: %w", cfg.From, err)
	}
	return &SMTP{
		addr:          cfg.Addr,
		host:          host,
		from:          from,
		username:      cfg.Username,
		password:      cfg.Password,
		allowInsecure: cfg.AllowInsecure,
		Now:           time.Now,
	}, nil
}

// errNoSTARTTLS refuses a server that would see the mail, links and
// credentials in plain text.
var errNoSTARTTLS = errors.New("the SMTP server doesn't offer STARTTLS; set SMTP_ALLOW_INSECURE=true only for a local mail catcher")

// Send implements Mailer. It gives up when ctx ends, or after 30 seconds.
func (s *SMTP) Send(ctx context.Context, m Message) (err error) {
	data, err := s.compose(m)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("smtp: dial %s: %w", s.addr, err)
	}
	defer func() {
		// Once ctx ends, the deadline below fails whatever was in flight;
		// say why instead of returning an i/o timeout.
		if err != nil && ctx.Err() != nil {
			err = fmt.Errorf("%w: %w", ctx.Err(), err)
		}
	}()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp: %w", err)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp: greeting from %s: %w", s.addr, err)
	}
	// Close only matters on the error paths; after Quit it has nothing to do.
	defer func() { _ = c.Close() }()

	if ok, _ := c.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}
		if s.TLS != nil {
			cfg = s.TLS.Clone()
			cfg.ServerName = s.host
		}
		if err := c.StartTLS(cfg); err != nil {
			return fmt.Errorf("smtp: starttls with %s: %w", s.addr, err)
		}
	} else if !s.allowInsecure {
		return errNoSTARTTLS
	}
	if s.username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.username, s.password.Reveal(), s.host)); err != nil {
			return fmt.Errorf("smtp: log in to %s: %w", s.addr, err)
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	if err := c.Rcpt(m.To); err != nil {
		return fmt.Errorf("smtp: RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("smtp: write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: end message: %w", err)
	}
	if err := c.Quit(); err != nil {
		return fmt.Errorf("smtp: QUIT: %w", err)
	}
	return nil
}

// compose builds the message: a multipart/alternative of the text and HTML
// parts, both quoted-printable UTF-8.
func (s *SMTP) compose(m Message) ([]byte, error) {
	for _, v := range []string{m.To, m.Subject} {
		if strings.ContainsAny(v, "\r\n") {
			return nil, errors.New("mail: a header value holds a line break")
		}
	}
	if _, err := mail.ParseAddress(m.To); err != nil {
		return nil, fmt.Errorf("mail: recipient %q: %w", m.To, err)
	}
	_, domain, _ := strings.Cut(s.from.Address, "@")

	var b bytes.Buffer
	body := multipart.NewWriter(&b)
	headers := []string{
		"From: " + s.from.String(),
		"To: " + m.To,
		"Subject: " + mime.QEncoding.Encode("utf-8", m.Subject),
		"Date: " + s.Now().Format(time.RFC1123Z),
		"Message-ID: <" + m.ID.String() + "@" + domain + ">",
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + body.Boundary(),
	}
	// The parts go after the headers, so write those first.
	var msg bytes.Buffer
	for _, h := range headers {
		msg.WriteString(h + "\r\n")
	}
	msg.WriteString("\r\n")

	for _, part := range []struct{ contentType, content string }{
		{"text/plain; charset=utf-8", m.Text},
		{"text/html; charset=utf-8", m.HTML},
	} {
		w, err := body.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {part.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, fmt.Errorf("mail: compose: %w", err)
		}
		qp := quotedprintable.NewWriter(w)
		if _, err := qp.Write([]byte(part.content)); err != nil {
			return nil, fmt.Errorf("mail: compose: %w", err)
		}
		if err := qp.Close(); err != nil {
			return nil, fmt.Errorf("mail: compose: %w", err)
		}
	}
	if err := body.Close(); err != nil {
		return nil, fmt.Errorf("mail: compose: %w", err)
	}
	msg.Write(b.Bytes())
	return msg.Bytes(), nil
}
