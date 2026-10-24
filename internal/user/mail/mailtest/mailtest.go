// Package mailtest has a Mailer for tests that keeps what it is sent.
package mailtest

import (
	"context"
	"slices"
	"sync"

	"github.com/vardius/go-api-boilerplate/internal/user/mail"
)

// Capture is a mail.Mailer that records messages instead of sending them.
// It is safe for concurrent use.
type Capture struct {
	mu    sync.Mutex
	sent  []mail.Message
	fails []error
}

var _ mail.Mailer = (*Capture)(nil)

// Send implements mail.Mailer. It returns the next error queued by
// FailNext, recording nothing, or else records m.
func (c *Capture) Send(_ context.Context, m mail.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.fails) > 0 {
		err := c.fails[0]
		c.fails = c.fails[1:]
		return err
	}
	c.sent = append(c.sent, m)
	return nil
}

// FailNext makes the next Send fail with err, after any failures already
// queued.
func (c *Capture) FailNext(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fails = append(c.fails, err)
}

// Sent returns the messages sent so far, oldest first.
func (c *Capture) Sent() []mail.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.sent)
}
