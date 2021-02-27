package mongo

import (
	"encoding/json"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

type JSONRawMessage json.RawMessage

func (m JSONRawMessage) MarshalBSON() ([]byte, error) {
	v, err := bson.Marshal(string(m))
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	return v, nil
}

func (m *JSONRawMessage) UnmarshalBSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var v string
	if err := bson.Unmarshal(data, &v); err != nil {
		return apperrors.Wrap(err)
	}

	if v != "" {
		*m = JSONRawMessage(v)
	}

	return nil
}
