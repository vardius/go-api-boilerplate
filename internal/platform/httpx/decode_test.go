package httpx_test

import (
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

type signUp struct {
	Email   string `json:"email"`
	Next    string `json:"next,omitzero"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	Tags []string `json:"tags"`
}

func jsonRequest(contentType, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/login-links", strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

func TestDecodeJSON(t *testing.T) {
	got, err := httpx.DecodeJSON[signUp](jsonRequest("application/json; charset=utf-8", `{"email":"ada@example.com","profile":{"name":"Ada"},"tags":["a"]}`), 1024)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "ada@example.com" || got.Profile.Name != "Ada" || len(got.Tags) != 1 {
		t.Errorf("decoded %+v", got)
	}
}

func TestDecodeJSONRejects(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		max         int64
		status      int
		message     string
		fields      map[string]string
	}{
		{"no content type", "", `{}`, 1024, 400, "The request body must be application/json.", nil},
		{"form", "application/x-www-form-urlencoded", `email=a`, 1024, 400, "The request body must be application/json.", nil},
		{"unknown field", "application/json", `{"email":"a","admin":true}`, 1024, 400, "", map[string]string{"admin": "is not a known field"}},
		{"unknown nested field", "application/json", `{"profile":{"role":"admin"}}`, 1024, 400, "", map[string]string{"profile.role": "is not a known field"}},
		{"field names are case-sensitive", "application/json", `{"Email":"a"}`, 1024, 400, "", map[string]string{"Email": "is not a known field"}},
		{"wrong field type", "application/json", `{"tags":[1]}`, 1024, 400, "", map[string]string{"tags.0": "has the wrong type"}},
		{"wrong top-level type", "application/json", `[]`, 1024, 400, "The request body has the wrong JSON type.", nil},
		{"trailing data", "application/json", `{"email":"a"} {"email":"b"}`, 1024, 400, "The request body is not valid JSON.", nil},
		{"duplicate field", "application/json", `{"email":"a","email":"b"}`, 1024, 400, "The request body is not valid JSON.", nil},
		{"truncated", "application/json", `{"email":`, 1024, 400, "The request body is not valid JSON.", nil},
		{"empty", "application/json", ``, 1024, 400, "The request body is empty.", nil},
		{"too large", "application/json", `{"email":"` + strings.Repeat("a", 64) + `"}`, 32, 413, "The request body is larger than 32 bytes.", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := httpx.DecodeJSON[signUp](jsonRequest(tt.contentType, tt.body), tt.max)
			if !errors.Is(err, apperr.ErrInvalid) {
				t.Fatalf("err = %v, want ErrInvalid", err)
			}
			if v.Email != "" {
				t.Errorf("returned a partly decoded value with the error: %+v", v)
			}
			if got := apperr.Message(err); got != tt.message {
				t.Errorf("message = %q, want %q", got, tt.message)
			}
			if tt.fields != nil {
				ve, ok := errors.AsType[*apperr.ValidationError](err)
				if !ok || !maps.Equal(ve.Fields, tt.fields) {
					t.Errorf("fields = %v, want %v", ve, tt.fields)
				}
			}
			if rec := serve(failWith(err), httptest.NewRequest(http.MethodPost, "/", nil)); rec.Code != tt.status {
				t.Errorf("WriteError status = %d, want %d", rec.Code, tt.status)
			}
		})
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("connection reset by peer") }
func (failingBody) Close() error             { return nil }

func TestDecodeJSONReadFailure(t *testing.T) {
	r := jsonRequest("application/json", "")
	r.Body = failingBody{}
	_, err := httpx.DecodeJSON[signUp](r, 1024)
	if !errors.Is(err, apperr.ErrInvalid) || apperr.Message(err) != "The request body could not be read." {
		t.Errorf("err = %v (%q)", err, apperr.Message(err))
	}
	if !strings.Contains(err.Error(), "connection reset by peer") {
		t.Errorf("the cause is lost for the logs: %v", err)
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		query  string
		want   httpx.Page
		fields map[string]string
	}{
		{"", httpx.Page{Limit: 20}, nil},
		{"?limit=50&cursor=MjAyNg", httpx.Page{Limit: 50, Cursor: "MjAyNg"}, nil},
		{"?limit=1", httpx.Page{Limit: 1}, nil},
		{"?limit=100", httpx.Page{Limit: 100}, nil},
		{"?limit=0", httpx.Page{}, map[string]string{"limit": "must be a whole number from 1 to 100"}},
		{"?limit=101", httpx.Page{}, map[string]string{"limit": "must be a whole number from 1 to 100"}},
		{"?limit=", httpx.Page{}, map[string]string{"limit": "must be a whole number from 1 to 100"}},
		{"?limit=ten&cursor=MjAyNg", httpx.Page{}, map[string]string{"limit": "must be a whole number from 1 to 100"}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, err := httpx.ParsePage(httptest.NewRequest(http.MethodGet, "/v1/clients"+tt.query, nil), 20, 100)
			if got != tt.want {
				t.Errorf("page = %+v, want %+v", got, tt.want)
			}
			if tt.fields == nil {
				if err != nil {
					t.Errorf("err = %v", err)
				}
				return
			}
			ve, ok := errors.AsType[*apperr.ValidationError](err)
			if !ok || !maps.Equal(ve.Fields, tt.fields) {
				t.Errorf("err = %v, want fields %v", err, tt.fields)
			}
		})
	}
}
