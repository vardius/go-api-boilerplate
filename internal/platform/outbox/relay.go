package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Handler delivers one message. Returning an error makes the relay retry
// the message later.
type Handler func(ctx context.Context, m Message) error

// RelayConfig tunes a Relay. Zero fields take the defaults shown.
type RelayConfig struct {
	// Interval is the wait between polls once no message is due.
	Interval time.Duration `env:"INTERVAL" envDefault:"1s"`
	// Lease is how long a claimed message stays claimed. It must outlast
	// Timeout, or a slow delivery can overlap its own retry.
	Lease time.Duration `env:"LEASE" envDefault:"2m"`
	// Timeout bounds one handler call.
	Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`
	// Batch is how many messages one claim takes.
	Batch int `env:"BATCH" envDefault:"10"`
	// MaxAttempts is how many times a message is tried before it is buried.
	MaxAttempts int `env:"MAX_ATTEMPTS" envDefault:"8"`
}

func (c RelayConfig) withDefaults() RelayConfig {
	if c.Interval <= 0 {
		c.Interval = time.Second
	}
	if c.Lease <= 0 {
		c.Lease = 2 * time.Minute
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.Batch <= 0 {
		c.Batch = 10
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 8
	}
	return c
}

// Relay delivers due messages from a Store to their topic's Handler.
type Relay struct {
	Store    Store
	Handlers map[string]Handler
	Config   RelayConfig
	// Logger defaults to slog.Default().
	Logger *slog.Logger
	// Now defaults to time.Now.
	Now func() time.Time

	initOnce  sync.Once
	delivered metric.Int64Counter
	buried    metric.Int64Counter
}

func (r *Relay) init() {
	r.initOnce.Do(func() {
		r.Config = r.Config.withDefaults()
		if r.Logger == nil {
			r.Logger = slog.Default()
		}
		if r.Now == nil {
			r.Now = time.Now
		}
		meter := otel.Meter("github.com/vardius/go-api-boilerplate/internal/platform/outbox")
		// The no-op meter never fails, and a failing real one returns a
		// usable no-op counter, so these errors carry nothing to act on.
		r.delivered, _ = meter.Int64Counter("outbox.delivered", metric.WithDescription("Messages delivered."))
		r.buried, _ = meter.Int64Counter("outbox.buried", metric.WithDescription("Messages given up on after repeated failures."))
	})
}

// Run delivers messages until ctx is done, then returns nil.
func (r *Relay) Run(ctx context.Context) error {
	r.init()
	ticker := time.NewTicker(r.Config.Interval)
	defer ticker.Stop()
	for {
		if _, err := r.Drain(ctx); err != nil && ctx.Err() == nil {
			r.Logger.ErrorContext(ctx, "outbox relay failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Drain delivers due messages until none is left and returns how many it
// handled, whatever the outcome of each.
func (r *Relay) Drain(ctx context.Context) (int, error) {
	r.init()
	handled := 0
	for {
		msgs, err := r.Store.Claim(ctx, r.Now(), r.Config.Lease, r.Config.Batch)
		if err != nil {
			return handled, fmt.Errorf("outbox: claim: %w", err)
		}
		for _, m := range msgs {
			r.deliver(ctx, m)
			handled++
		}
		if len(msgs) < r.Config.Batch {
			return handled, nil
		}
	}
}

func (r *Relay) deliver(ctx context.Context, m Message) {
	log := r.Logger.With("message_id", m.ID.String(), "topic", m.Topic, "attempt", m.Attempts)

	h, ok := r.Handlers[m.Topic]
	if !ok {
		r.bury(ctx, log, m, "no handler for topic "+m.Topic)
		return
	}
	// A message claimed more often than allowed was in flight when a relay
	// crashed each time; it may be what crashes it.
	if m.Attempts > r.Config.MaxAttempts {
		r.bury(ctx, log, m, fmt.Sprintf("claimed %d times without finishing", m.Attempts))
		return
	}

	err := r.call(ctx, h, m)
	if err == nil {
		if err := r.Store.Ack(ctx, m.ID); err != nil {
			log.ErrorContext(ctx, "outbox ack failed; the message will be delivered again", "err", err)
			return
		}
		r.delivered.Add(ctx, 1, metric.WithAttributes(attribute.String("topic", m.Topic)))
		return
	}
	if m.Attempts >= r.Config.MaxAttempts {
		r.bury(ctx, log, m, err.Error())
		return
	}

	retryAt := r.Now().Add(Backoff(m.Attempts))
	log.WarnContext(ctx, "outbox delivery failed; retrying", "err", err, "retry_at", retryAt)
	if err := r.Store.Retry(ctx, m.ID, retryAt, err.Error()); err != nil {
		log.ErrorContext(ctx, "outbox retry failed; the message will be claimed again when its lease ends", "err", err)
	}
}

// call runs h, turning a panic into an error so that one bad message can't
// stop delivery of the rest.
func (r *Relay) call(ctx context.Context, h Handler, m Message) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.Config.Timeout)
	defer cancel()
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("handler panicked: %v", v)
		}
	}()
	return h(ctx, m)
}

func (r *Relay) bury(ctx context.Context, log *slog.Logger, m Message, reason string) {
	log.ErrorContext(ctx, "outbox message buried", "reason", reason)
	if err := r.Store.Bury(ctx, m.ID, reason); err != nil {
		log.ErrorContext(ctx, "outbox bury failed", "err", err)
		return
	}
	r.buried.Add(ctx, 1, metric.WithAttributes(attribute.String("topic", m.Topic)))
}

// Backoff is the wait before retrying a message that failed on its
// attempt-th try: one second, doubling each time, at most five minutes.
func Backoff(attempt int) time.Duration {
	const ceiling = 5 * time.Minute
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 10 {
		return ceiling
	}
	return min(time.Second<<(attempt-1), ceiling)
}
