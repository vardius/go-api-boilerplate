package user

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestUnmarshalChangeEmailAddress(t *testing.T) {
	testJSON := []byte(`{"id":"4dded431-acee-4078-86c6-9dffa5efba1e","email":"test@test.com"}`)

	testUnmarshalCommand(t, testJSON, &ChangeEmailAddress{})
}

func TestUnmarshalRegisterWithEmail(t *testing.T) {
	testJSON := []byte(`{"email":"test@test.com"}`)

	testUnmarshalCommand(t, testJSON, &RegisterWithEmail{})
}

func TestUnmarshalRegisterWithFacebook(t *testing.T) {
	testJSON := []byte(`{"email":"test@test.com","facebook_id":"","access_token":""}`)

	testUnmarshalCommand(t, testJSON, &RegisterWithFacebook{})
}

func TestUnmarshalRegisterWithGoogle(t *testing.T) {
	testJSON := []byte(`{"email":"test@test.com","google_id":"","access_token":""}`)

	testUnmarshalCommand(t, testJSON, &RegisterWithGoogle{})
}

func testUnmarshalCommand(t *testing.T, testJSON []byte, c interface{}) {
	if err := json.Unmarshal(testJSON, c); err != nil {
		t.Fatal(err)
	}

	j, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	cmp := bytes.Compare(j, testJSON)
	if cmp != 0 {
		t.Errorf("Serialize command did not match expected result: %s | %s | %d", string(j), string(testJSON), cmp)
	}
}
