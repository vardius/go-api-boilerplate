package user

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestEmailAddress_UnmarshalJSON(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		e       EmailAddress
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "valid email", e: "", args: args{[]byte(`"example@test.com"`)}, wantErr: false},
		{name: "invalid email", e: "", args: args{[]byte(`"not an email"`)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.UnmarshalJSON(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmailAddress_MarshalJSON_WasRegisteredWithEmail(t *testing.T) {
	tests := []struct {
		name    string
		e       EmailAddress
		want    []byte
		wantErr bool
	}{
		{name: "valid email", e: "example@test.com", want: []byte(`{"id":"ec372f56-5937-4600-92f1-2115fa239d52","email":"example@test.com"}`), wantErr: false},
		{name: "invalid email", e: "not an email", want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := WasRegisteredWithEmail{
				ID:    uuid.MustParse("ec372f56-5937-4600-92f1-2115fa239d52"),
				Email: tt.e,
			}
			got, err := json.Marshal(e)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %s, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %s, want %s", got, tt.want)
			}
		})
	}
}
