package mongo

import (
	"encoding/json"
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
