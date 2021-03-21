package mongo

import (
	"encoding/json"
	"errors"
	"fmt"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type JSONRawMessage json.RawMessage

func (m JSONRawMessage) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, string(m)), nil
}

func (m *JSONRawMessage) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if t != bsontype.String {
		return apperrors.Wrap(fmt.Errorf("invalid type: %s", t))
	}

	str, _, ok := bsoncore.ReadString(data)
	if ok {
		*m = JSONRawMessage(str)
	}

	return nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (m JSONRawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *JSONRawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}
