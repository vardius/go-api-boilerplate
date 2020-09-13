package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

// NullInt64 is an alias for sql.NullInt64 data type
type NullInt64 struct{ sql.NullInt64 }

// MarshalJSON for NullInt64
func (ni NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}

	jsn, err := json.Marshal(ni.Int64)
	if err != nil {
		return jsn, fmt.Errorf("MySQL could not marshal NullInt64: %w", err)
	}

	return jsn, nil
}

// UnmarshalJSON for NullInt64
func (ni NullInt64) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &ni.Int64); err != nil {
		return apperrors.Wrap(fmt.Errorf("MySQL NullInt64 unmarshal error: %w", err))
	}

	ni.Valid = true

	return nil
}

// NullBool is an alias for sql.NullBool data type
type NullBool struct{ sql.NullBool }

// MarshalJSON for NullBool
func (nb NullBool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return []byte("null"), nil
	}

	jsn, err := json.Marshal(nb.Bool)
	if err != nil {
		return jsn, fmt.Errorf("MySQL could not marshal NullBool: %w", err)
	}

	return jsn, nil
}

// UnmarshalJSON for NullBool
func (nb NullBool) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &nb.Bool); err != nil {
		return apperrors.Wrap(fmt.Errorf("MySQL NullBool unmarshal error: %w", err))
	}

	nb.Valid = true

	return nil
}

// NullFloat64 is an alias for sql.NullFloat64 data type
type NullFloat64 struct{ sql.NullFloat64 }

// MarshalJSON for NullFloat64
func (nf NullFloat64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}

	jsn, err := json.Marshal(nf.Float64)
	if err != nil {
		return jsn, fmt.Errorf("MySQL could not marshal NullFloat64: %w", err)
	}

	return jsn, nil
}

// UnmarshalJSON for NullFloat64
func (nf NullFloat64) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &nf.Float64); err != nil {
		return apperrors.Wrap(fmt.Errorf("MySQL NullFloat64 unmarshal error: %w", err))
	}

	nf.Valid = true

	return nil
}

// NullString is an alias for sql.NullString data type
type NullString struct{ sql.NullString }

// MarshalJSON for NullString
func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}

	jsn, err := json.Marshal(ns.String)
	if err != nil {
		return jsn, fmt.Errorf("MySQL could not marshal NullString: %w", err)
	}

	return jsn, nil
}

// UnmarshalJSON for NullString
func (ns NullString) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &ns.String); err != nil {
		return apperrors.Wrap(fmt.Errorf("MySQL NullString unmarshal error: %w", err))
	}

	ns.Valid = true

	return nil
}

// NullTime is an alias for mysql.NullTime data type
type NullTime struct{ sql.NullTime }

// MarshalJSON for NullTime
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	val := fmt.Sprintf("\"%s\"", nt.Time.Format(time.RFC3339))
	return []byte(val), nil
}

// UnmarshalJSON for NullTime
func (nt NullTime) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &nt.Time); err != nil {
		return apperrors.Wrap(fmt.Errorf("MySQL NullTime unmarshal error: %w", err))
	}

	nt.Valid = true

	return nil
}
