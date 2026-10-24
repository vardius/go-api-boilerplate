// Package estest helps test event-sourced code: Given/When/Then for
// aggregates, and TestStore, the contract every es.Store must meet.
package estest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
)

// Scenario is one aggregate test: its history, the action, and the outcome.
//
//	estest.Given(t, user.New(), user.Registered{…}).
//		When(func(u *user.User) error { return u.ChangeEmail(…) }).
//		Then(user.EmailChanged{…})
type Scenario[A es.Aggregate] struct {
	t   testing.TB
	a   A
	err error
}

// Given replays history into a, a new aggregate.
func Given[A es.Aggregate](t testing.TB, a A, history ...any) *Scenario[A] {
	t.Helper()
	if err := es.Replay(a, history...); err != nil {
		t.Fatalf("replay history: %v", err)
	}
	return &Scenario[A]{t: t, a: a}
}

// When runs the action under test on the aggregate.
func (s *Scenario[A]) When(action func(A) error) *Scenario[A] {
	s.err = action(s.a)
	return s
}

// Then checks that the action succeeded and raised exactly want, in order.
func (s *Scenario[A]) Then(want ...any) A {
	s.t.Helper()
	if s.err != nil {
		s.t.Fatalf("action failed: %v", s.err)
	}
	got := es.Pending(s.a)
	if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
		s.t.Fatalf("raised events:\n got %#v\nwant %#v", got, want)
	}
	return s.a
}

// ThenError checks that the action failed with an error matching target
// (errors.Is) and raised nothing.
func (s *Scenario[A]) ThenError(target error) {
	s.t.Helper()
	if !errors.Is(s.err, target) {
		s.t.Fatalf("error = %v, want %v", s.err, target)
	}
	if got := es.Pending(s.a); len(got) > 0 {
		s.t.Fatalf("a failed action raised %#v", got)
	}
}
