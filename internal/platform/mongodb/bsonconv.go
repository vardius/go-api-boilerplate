package mongodb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// uuid.UUID is a plain [16]byte, which the driver would store as an array
// of 16 numbers, so documents hold UUIDs through these two functions.

func binaryUUID(id uuid.UUID) bson.Binary {
	return bson.Binary{Subtype: bson.TypeBinaryUUID, Data: id[:]}
}

func parseBinaryUUID(b bson.Binary) (uuid.UUID, error) {
	if b.Subtype != bson.TypeBinaryUUID || len(b.Data) != len(uuid.UUID{}) {
		return uuid.UUID{}, fmt.Errorf("want a UUID, got %d bytes of binary subtype %#x", len(b.Data), b.Subtype)
	}
	return uuid.UUID(b.Data), nil
}

// jsonToBSON converts a JSON object to a BSON document with the same keys in
// the same order. Integers become int32 or int64, or decimal128 when they
// don't fit an int64; other numbers become doubles. Keys are kept as they
// are, so "$date" is a key, not Extended JSON.
func jsonToBSON(data []byte) (bson.D, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if tok != json.Delim('{') {
		return nil, errors.New("decode JSON: not an object")
	}
	doc, err := decodeObject(dec)
	if err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("decode JSON: data after the object")
	}
	return doc, nil
}

// decodeObject reads an object's members after its opening brace.
func decodeObject(dec *json.Decoder) (bson.D, error) {
	doc := bson.D{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("object key %v is not a string", tok)
		}
		value, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		doc = append(doc, bson.E{Key: key, Value: value})
	}
	_, err := dec.Token() // the closing brace
	return doc, err
}

// decodeArray reads an array's elements after its opening bracket.
func decodeArray(dec *json.Decoder) (bson.A, error) {
	arr := bson.A{}
	for dec.More() {
		value, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		arr = append(arr, value)
	}
	_, err := dec.Token() // the closing bracket
	return arr, err
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		if v == '{' {
			return decodeObject(dec)
		}
		// Token reports a misplaced closing delimiter as an error, so this
		// is an opening bracket.
		return decodeArray(dec)
	case json.Number:
		return bsonNumber(v)
	default: // string, bool or nil
		return v, nil
	}
}

func bsonNumber(n json.Number) (any, error) {
	s := n.String()
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("number %s: %w", s, err)
		}
		return f, nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		if i >= math.MinInt32 && i <= math.MaxInt32 {
			return int32(i), nil
		}
		return i, nil
	}
	d, err := bson.ParseDecimal128(s)
	if err != nil {
		return nil, fmt.Errorf("integer %s has more digits than decimal128 keeps", s)
	}
	return d, nil
}

// bsonToJSON converts a document written by jsonToBSON back to JSON. It
// fails on BSON types that JSON has no value for.
func bsonToJSON(doc bson.D) ([]byte, error) {
	var b bytes.Buffer
	if err := writeJSON(&b, doc); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeJSON(b *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case bson.D:
		b.WriteByte('{')
		for i, e := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, e.Key)
			b.WriteByte(':')
			if err := writeJSON(b, e.Value); err != nil {
				return fmt.Errorf("%q: %w", e.Key, err)
			}
		}
		b.WriteByte('}')
	case bson.A:
		b.WriteByte('[')
		for i, elem := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeJSON(b, elem); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		b.WriteByte(']')
	case string:
		writeString(b, v)
	case int32:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%v has no JSON value", v)
		}
		// Marshal writes the shortest form that reads back as v.
		data, _ := json.Marshal(v)
		b.Write(data)
	case bson.Decimal128:
		if v.IsNaN() || v.IsInf() != 0 {
			return fmt.Errorf("%v has no JSON value", v)
		}
		b.WriteString(v.String())
	case bool:
		b.WriteString(strconv.FormatBool(v))
	case nil:
		b.WriteString("null")
	default:
		return fmt.Errorf("BSON %T has no JSON value", v)
	}
	return nil
}

func writeString(b *bytes.Buffer, s string) {
	// Marshaling a string can't fail.
	data, _ := json.Marshal(s)
	b.Write(data)
}
