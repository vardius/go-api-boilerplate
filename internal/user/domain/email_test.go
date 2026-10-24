package domain_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
)

func TestParseEmail(t *testing.T) {
	local64 := strings.Repeat("a", 64)
	// 64 + 1 + 189 = 254 bytes, the longest address allowed.
	longest := local64 + "@" + strings.Repeat("b", 185) + ".com"
	tests := []struct {
		in, want string
	}{
		{"ada@example.com", "ada@example.com"},
		{"Ada.Lovelace@Example.COM", "ada.lovelace@example.com"},
		{"  ada@example.com\t", "ada@example.com"},
		{"ada+tag@sub.example.co.uk", "ada+tag@sub.example.co.uk"},
		{"zażółć@example.com", "zażółć@example.com"},
		{"ŁUKASZ@example.com", "łukasz@example.com"},
		{longest, longest},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := domain.ParseEmail(tt.in)
			if err != nil {
				t.Fatalf("ParseEmail(%q): %v", tt.in, err)
			}
			if got.String() != tt.want {
				t.Errorf("ParseEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseEmailRejects(t *testing.T) {
	tests := []struct {
		name, in, problem string
	}{
		{"empty", "", "is required"},
		{"blank", "   ", "is required"},
		{"no at", "ada.example.com", "is not an email address"},
		{"no domain", "ada@", "is not an email address"},
		{"display name", "Ada <ada@example.com>", "is not an email address"},
		{"angle brackets", "<ada@example.com>", "is not an email address"},
		{"two addresses", "ada@example.com, bob@example.com", "is not an email address"},
		{"quoted local part", `"ada lovelace"@example.com`, "is not an email address"},
		{"comment", "ada@example.com (Ada)", "is not an email address"},
		{"header injection", "ada@example.com\r\nBcc: eve@example.com", "is not an email address"},
		{"over 254 bytes", strings.Repeat("a", 64) + "@" + strings.Repeat("b", 186) + ".com", "is longer than 254 bytes"},
		// Ⱥ is two bytes and its lower case, ⱥ, three: the limit holds after.
		{"over 254 bytes once lower-cased", strings.Repeat("Ⱥ", 60) + "@" + strings.Repeat("b", 129) + ".com", "is longer than 254 bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.ParseEmail(tt.in)
			v, ok := errors.AsType[*apperr.ValidationError](err)
			if !ok || v.Fields["email"] != tt.problem {
				t.Fatalf("ParseEmail(%q): err = %v, want email %q", tt.in, err, tt.problem)
			}
		})
	}
}

func TestEmailText(t *testing.T) {
	var zero domain.Email
	if !zero.IsZero() || zero.String() != "" {
		t.Errorf("zero Email = %q", zero)
	}
	data, err := json.Marshal(struct{ E domain.Email }{mustEmail("ada@example.com")})
	if err != nil || string(data) != `{"E":"ada@example.com"}` {
		t.Fatalf("marshal = %s, %v", data, err)
	}
	// A stored address reads back as written, even one today's rules reject.
	var back struct{ E domain.Email }
	if err := json.Unmarshal([]byte(`{"E":"Legacy Address"}`), &back); err != nil || back.E.String() != "Legacy Address" {
		t.Errorf("unmarshal = %q, %v", back.E, err)
	}
}

func FuzzParseEmail(f *testing.F) {
	for _, s := range []string{"ada@example.com", "Ada <ada@example.com>", "a@b", "İ@example.com", `"x"@y.z`, "a@[127.0.0.1]", "@", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		e, err := domain.ParseEmail(s)
		if err != nil {
			if !errors.Is(err, apperr.ErrInvalid) {
				t.Fatalf("ParseEmail(%q): err = %v, want an apperr.ErrInvalid", s, err)
			}
			return
		}
		got := e.String()
		switch {
		case e.IsZero():
			t.Fatalf("ParseEmail(%q) accepted no address", s)
		case len(got) > 254:
			t.Fatalf("ParseEmail(%q) = %d bytes", s, len(got))
		case !utf8.ValidString(got):
			t.Fatalf("ParseEmail(%q) = %q, not UTF-8", s, got)
		case strings.ToLower(got) != got:
			t.Fatalf("ParseEmail(%q) = %q, not lower-case", s, got)
		case strings.ContainsAny(got, "\r\n<>, "):
			t.Fatalf("ParseEmail(%q) = %q, which could break a header", s, got)
		case !strings.Contains(got, "@"):
			t.Fatalf("ParseEmail(%q) = %q, no @", s, got)
		}
		again, err := domain.ParseEmail(got)
		if err != nil || again != e {
			t.Fatalf("ParseEmail(%q) = %q, which parses again as %q, %v", s, got, again, err)
		}
	})
}
