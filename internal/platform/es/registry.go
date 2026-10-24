package es

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Registry maps event type names to payload types. Build one per service at
// startup and pass it where events are encoded or decoded.
type Registry struct {
	types     map[string]reflect.Type
	names     map[reflect.Type]string
	upcasters map[string]upcaster
}

type upcaster struct {
	to string
	fn func([]byte) ([]byte, error)
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		types:     map[string]reflect.Type{},
		names:     map[reflect.Type]string{},
		upcasters: map[string]upcaster{},
	}
}

// Register names the payload type T. Names are stored with every event, so
// they never change: "user.registered.v1". A new shape gets a new name and
// an Upcast from the old one.
//
// Register panics on an empty or repeated name, a type registered twice, or
// a T that isn't a struct, because each is a programming error.
func Register[T any](r *Registry, name string) {
	t := reflect.TypeFor[T]()
	switch {
	case name == "":
		panic("es: Register with an empty name")
	case t.Kind() != reflect.Struct:
		panic(fmt.Sprintf("es: Register %q: payload must be a struct, not %s", name, t))
	}
	if _, ok := r.types[name]; ok {
		panic(fmt.Sprintf("es: event type %q is already registered", name))
	}
	if _, ok := r.upcasters[name]; ok {
		panic(fmt.Sprintf("es: event type %q is upcast to another type", name))
	}
	if other, ok := r.names[t]; ok {
		panic(fmt.Sprintf("es: %s is already registered as %q", t, other))
	}
	r.types[name] = t
	r.names[t] = name
}

// Upcast reads events stored as from as events of type to, converting the
// JSON payload with fn. It lets a payload change shape while the stored
// events stay as they were written. Upcasts may chain.
func (r *Registry) Upcast(from, to string, fn func(payload []byte) ([]byte, error)) {
	if _, ok := r.types[from]; ok {
		panic(fmt.Sprintf("es: Upcast %q: it is a registered type", from))
	}
	if _, ok := r.upcasters[from]; ok {
		panic(fmt.Sprintf("es: Upcast %q: already upcast", from))
	}
	r.upcasters[from] = upcaster{to: to, fn: fn}
}

// Encode returns the registered name of payload's type and the payload as
// JSON.
func (r *Registry) Encode(payload any) (string, []byte, error) {
	name, ok := r.names[reflect.TypeOf(payload)]
	if !ok {
		return "", nil, fmt.Errorf("es: %T is not a registered event type", payload)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("es: encode %s: %w", name, err)
	}
	return name, data, nil
}

// Decode returns the payload stored under name, upcast to its current type.
// An unknown name returns ErrUnknownEventType.
func (r *Registry) Decode(name string, data []byte) (any, error) {
	for seen := 0; ; seen++ {
		u, ok := r.upcasters[name]
		if !ok {
			break
		}
		if seen > len(r.upcasters) {
			return nil, fmt.Errorf("es: upcasts from %q form a loop", name)
		}
		upcast, err := u.fn(data)
		if err != nil {
			return nil, fmt.Errorf("es: upcast %s to %s: %w", name, u.to, err)
		}
		name, data = u.to, upcast
	}

	t, ok := r.types[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownEventType, name)
	}
	v := reflect.New(t)
	if err := json.Unmarshal(data, v.Interface()); err != nil {
		return nil, fmt.Errorf("es: decode %s: %w", name, err)
	}
	return v.Elem().Interface(), nil
}
