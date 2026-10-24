package store_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
)

type purger struct {
	mu    sync.Mutex
	times []time.Time
	err   error
}

func (p *purger) Purge(_ context.Context, now time.Time) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.times = append(p.times, now)
	return p.err
}

func (p *purger) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.times)
}

func TestPurgeWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer
		p := &purger{err: errors.New("database is down")}
		w := store.PurgeWorker(p, 10*time.Minute, time.Now, slog.New(slog.NewTextHandler(&logs, nil)))
		served := make(chan error, 1)
		go func() { served <- w.Serve() }()

		time.Sleep(25 * time.Minute)
		synctest.Wait()
		if n := p.calls(); n != 2 {
			t.Errorf("purged %d times in 25 minutes, want 2", n)
		}
		if strings.Count(logs.String(), "database is down") != 2 {
			t.Errorf("failures weren't logged: %q", logs.String())
		}

		if err := w.Shutdown(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := <-served; err != nil {
			t.Errorf("Serve = %v after Shutdown", err)
		}
	})
}

func TestHashToken(t *testing.T) {
	// sha256("abc") from FIPS 180-2, Appendix B.1.
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	h := store.HashToken("abc")
	if got := hex.EncodeToString(h[:]); got != want {
		t.Errorf("HashToken(abc) = %s, want %s", got, want)
	}
}
