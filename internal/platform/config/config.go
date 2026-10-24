// Package config loads a service's configuration from environment variables
// and documents it from the same source.
//
// A configuration is a struct whose fields carry caarlos0/env tags: env names
// the variable (with the options required and notEmpty), envDefault gives
// its default, and envPrefix nests a platform struct under a prefix, as in
// `envPrefix:"HTTP_"` on an httpx.ServerConfig. Every field also has a desc
// tag saying what it is for. Load reads the struct; Describe and Markdown
// turn the same tags into the configuration reference, so the code and the
// docs can't drift. Credentials have type Secret, which never prints.
package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Load reads a T from the environment. Its error names every variable that
// is missing or doesn't parse, not just the first.
func Load[T any]() (T, error) {
	cfg, err := env.ParseAs[T]()
	if err != nil {
		var zero T
		return zero, &loadError{problems: problems(fields(reflect.TypeFor[T]()), err), err: err}
	}
	return cfg, nil
}

type loadError struct {
	problems []string
	err      error
}

func (e *loadError) Error() string { return "config: " + strings.Join(e.problems, "; ") }

func (e *loadError) Unwrap() error { return e.err }

// problems rewrites env's errors to name variables. env reports a value that
// doesn't parse by its Go field name, which two nested structs can share, so
// it is matched back to every variable that field could be.
func problems(all []field, err error) []string {
	errs := []error{err}
	if agg, ok := errors.AsType[env.AggregateError](err); ok {
		errs = agg.Errors
	}
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		switch e := e.(type) {
		case env.VarIsNotSetError:
			out = append(out, e.Key+" is required")
		case env.EmptyVarError:
			out = append(out, e.Key+" must not be empty")
		case env.ParseError:
			var names []string
			for _, f := range all {
				if f.goName == e.Name && f.goType == e.Type {
					names = append(names, f.Name)
				}
			}
			if len(names) == 0 {
				out = append(out, e.Error())
				continue
			}
			out = append(out, strings.Join(names, " or ")+": "+e.Err.Error())
		default:
			out = append(out, e.Error())
		}
	}
	return out
}

const redacted = "[redacted]"

// Secret is a string that never shows its value: printing, logging and
// encoding it as JSON or text all give "[redacted]". Read it with Reveal at
// the one place that needs it.
type Secret string

// Reveal returns the secret's value.
func (s Secret) Reveal() string { return string(s) }

// String returns "[redacted]".
func (Secret) String() string { return redacted }

// GoString returns "[redacted]", for %#v.
func (Secret) GoString() string { return redacted }

// Format writes "[redacted]" for every verb. Without it, verbs that don't
// consult String, such as %d, would print the value.
func (Secret) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, redacted) }

// LogValue returns "[redacted]" to slog.
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON returns the JSON string "[redacted]".
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// MarshalText returns "[redacted]".
func (Secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }

// Var documents one environment variable.
type Var struct {
	Name    string
	Default string
	Desc    string
	// Required is set for the env options required and notEmpty.
	Required bool
	// Secret is set for fields of type Secret.
	Secret bool
}

// Describe lists the variables a T reads, in field order, nested structs
// included under their prefixes.
func Describe[T any]() []Var {
	all := fields(reflect.TypeFor[T]())
	vars := make([]Var, len(all))
	for i, f := range all {
		vars[i] = f.Var
	}
	return vars
}

type field struct {
	Var
	goName string
	goType reflect.Type
}

func fields(t reflect.Type) []field {
	var out []field
	if t.Kind() == reflect.Struct {
		walk(t, "", &out)
	}
	return out
}

// walk visits fields the way env.Parse does: it recurses into anonymous
// structs, into struct fields (under their envPrefix) and into pointers to
// structs marked init, which are the only pointers set on a zero value.
func walk(t reflect.Type, prefix string, out *[]field) {
	secret := reflect.TypeFor[Secret]()
	for sf := range t.Fields() {
		if !sf.IsExported() {
			continue
		}
		nested := prefix + sf.Tag.Get("envPrefix")
		if sf.Type.Kind() == reflect.Struct && sf.Type.Name() == "" {
			walk(sf.Type, nested, out)
			continue
		}
		key, opts, _ := strings.Cut(sf.Tag.Get("env"), ",")
		options := strings.Split(opts, ",")
		if key == "-" || slices.Contains(options, "-") {
			continue
		}
		if key != "" {
			def := sf.Tag.Get("envDefault")
			*out = append(*out, field{
				Name:     prefix + key,
				Default:  def,
				Desc:     sf.Tag.Get("desc"),
				Required: slices.Contains(options, "required") || slices.Contains(options, "notEmpty"),
				Secret:   sf.Type == secret || sf.Type.Kind() == reflect.Pointer && sf.Type.Elem() == secret,
				goName:   sf.Name,
				goType:   sf.Type,
			})
		}
		switch {
		case sf.Type.Kind() == reflect.Struct:
			walk(sf.Type, nested, out)
		case sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct && slices.Contains(options, "init"):
			walk(sf.Type.Elem(), nested, out)
		}
	}
}

// Markdown renders vars as a Markdown table for the configuration reference.
func Markdown(vars []Var) string {
	var b strings.Builder
	b.WriteString("| Variable | Default | Required | Description |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, v := range vars {
		def := ""
		if v.Default != "" {
			def = "`" + cell(v.Default) + "`"
		}
		required := ""
		if v.Required {
			required = "yes"
		}
		desc := cell(v.Desc)
		if v.Secret {
			desc = strings.TrimSpace("Secret. " + desc)
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", v.Name, def, required, desc)
	}
	return b.String()
}

func cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.Join(strings.Fields(s), " ")
}
