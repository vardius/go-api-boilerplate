package mongodb

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// roundTrip converts payload the way the event store does: to a document,
// through BSON bytes inside a parent document, and back to JSON.
func roundTrip(t *testing.T, payload string) string {
	t.Helper()
	doc, err := jsonToBSON([]byte(payload))
	if err != nil {
		t.Fatalf("jsonToBSON(%s): %v", payload, err)
	}
	data, err := bson.Marshal(bson.D{{Key: "payload", Value: doc}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var stored struct {
		Payload bson.D `bson:"payload"`
	}
	if err := bson.Unmarshal(data, &stored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := bsonToJSON(stored.Payload)
	if err != nil {
		t.Fatalf("bsonToJSON: %v", err)
	}
	return string(out)
}

func TestPayloadRoundTrip(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty object", `{}`, `{}`},
		{"key order kept", `{"z":1,"a":2,"m":3}`, `{"z":1,"a":2,"m":3}`},
		{"unicode", `{"text":"zażółć ✓","emoji":"🙂"}`, `{"text":"zażółć ✓","emoji":"🙂"}`},
		{"escapes", `{"s":"a\"b\\c\n\u0001"}`, `{"s":"a\"b\\c\n\u0001"}`},
		{"nested", `{"a":{"b":{"c":[1,[2,{}],[]]}}}`, `{"a":{"b":{"c":[1,[2,{}],[]]}}}`},
		{"literals", `{"t":true,"f":false,"n":null}`, `{"t":true,"f":false,"n":null}`},
		{"int32 bounds", `{"min":-2147483648,"max":2147483647}`, `{"min":-2147483648,"max":2147483647}`},
		{"int64 bounds", `{"min":-9223372036854775808,"max":9223372036854775807}`, `{"min":-9223372036854775808,"max":9223372036854775807}`},
		{"uint64 max", `{"n":18446744073709551615}`, `{"n":18446744073709551615}`},
		{"floats", `{"a":1.5,"b":-0.001,"c":1e+21,"d":1e-7}`, `{"a":1.5,"b":-0.001,"c":1e+21,"d":1e-7}`},
		{"float spelled as integer", `{"a":1.0,"b":2E2}`, `{"a":1,"b":200}`},
		{"extended JSON keys are plain keys", `{"$date":"2026","$oid":1,"a.b":{"$numberLong":"5"}}`, `{"$date":"2026","$oid":1,"a.b":{"$numberLong":"5"}}`},
		{"whitespace dropped", " {\n \"a\" : [ 1 , 2 ] }\n", `{"a":[1,2]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roundTrip(t, tt.in); got != tt.want {
				t.Errorf("round trip of %s = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestJSONNumbersBecomeBSONNumbers(t *testing.T) {
	doc, err := jsonToBSON([]byte(`{"small":7,"large":4294967296,"huge":18446744073709551615,"real":0.5}`))
	if err != nil {
		t.Fatal(err)
	}
	huge, err := bson.ParseDecimal128("18446744073709551615")
	if err != nil {
		t.Fatal(err)
	}
	want := bson.D{
		{Key: "small", Value: int32(7)},
		{Key: "large", Value: int64(4294967296)},
		{Key: "huge", Value: huge},
		{Key: "real", Value: 0.5},
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("jsonToBSON = %#v, want %#v", doc, want)
	}
}

func TestJSONToBSONRejects(t *testing.T) {
	tests := []struct {
		name, in, wantErr string
	}{
		{"empty", ``, "decode JSON"},
		{"array", `[1]`, "not an object"},
		{"string", `"x"`, "not an object"},
		{"malformed", `{"a":}`, "decode JSON"},
		{"unclosed", `{"a":[1,2}`, "decode JSON"},
		{"trailing data", `{"a":1}{"b":2}`, "data after the object"},
		{"float overflow", `{"a":1e400}`, "number 1e400"},
		{"integer past decimal128", `{"a":` + strings.Repeat("9", 40) + `}`, "more digits than decimal128 keeps"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jsonToBSON([]byte(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("jsonToBSON(%s): err = %v, want one containing %q", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestBSONToJSONRejectsTypesJSONLacks(t *testing.T) {
	nan, err := bson.ParseDecimal128("NaN")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		doc     bson.D
		wantErr string
	}{
		{"date", bson.D{{Key: "at", Value: bson.DateTime(0)}}, `"at": BSON bson.DateTime`},
		{"binary in an array", bson.D{{Key: "a", Value: bson.A{int32(1), bson.Binary{}}}}, `"a": [1]: BSON bson.Binary`},
		{"NaN double", bson.D{{Key: "x", Value: math.NaN()}}, `"x": NaN`},
		{"infinite double", bson.D{{Key: "x", Value: math.Inf(-1)}}, `"x": -Inf`},
		{"NaN decimal", bson.D{{Key: "x", Value: nan}}, `"x": NaN`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bsonToJSON(tt.doc)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("bsonToJSON: err = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestBinaryUUID(t *testing.T) {
	id := uuid.MustParse("01920000-0000-7000-8000-00000000abcd")
	b := binaryUUID(id)
	if b.Subtype != 0x04 || string(b.Data) != string(id[:]) {
		t.Fatalf("binaryUUID = %#v, want subtype 4 holding the UUID's bytes", b)
	}
	got, err := parseBinaryUUID(b)
	if err != nil || got != id {
		t.Fatalf("parseBinaryUUID = %s, %v; want %s", got, err, id)
	}

	for _, bad := range []bson.Binary{
		{Subtype: 0x00, Data: id[:]},
		{Subtype: 0x03, Data: id[:]},
		{Subtype: 0x04, Data: id[:15]},
		{},
	} {
		if _, err := parseBinaryUUID(bad); err == nil {
			t.Errorf("parseBinaryUUID(%#v) accepted it", bad)
		}
	}
}
