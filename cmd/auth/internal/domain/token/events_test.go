package token

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestWasCreated_TokenInfo(t *testing.T) {
	file, err := os.Open("events_test_was_created.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	payload, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		body    json.RawMessage
		wantErr bool
	}{
		{"payload", payload, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event WasCreated
			if err := json.Unmarshal(tt.body, &event); (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal event payload error = %v, wantErr %v", err, tt.wantErr)
			}
			_, err := event.TokenInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
