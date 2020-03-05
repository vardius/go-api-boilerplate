package user

import "testing"

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
		{name: "valid email", e: "", args: args{[]byte(`"not an email"`)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.UnmarshalJSON(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
