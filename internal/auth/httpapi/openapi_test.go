package httpapi_test

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
)

func TestOpenAPIDocumentIsValid(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.1") {
		t.Errorf("openapi = %s, want 3.1", doc.OpenAPI)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatal(err)
	}
}

// The document lists exactly the routes the router serves.
func TestOpenAPIDocumentsEveryRoute(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	documented := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			documented[method+" "+path] = true
		}
	}
	served := map[string]bool{}
	for _, route := range newRig(t, options{}).router.Routes() {
		served[route.Method+" "+route.Pattern] = true
	}
	for _, route := range slices.Sorted(maps.Keys(served)) {
		if !documented[route] {
			t.Errorf("%s is served but not documented", route)
		}
	}
	for _, route := range slices.Sorted(maps.Keys(documented)) {
		if !served[route] {
			t.Errorf("%s is documented but not served", route)
		}
	}
}

// Every error a route documents under /oauth/token and /oauth/revoke is
// RFC 6749's, not a problem, except the ones httpx answers itself.
func TestOAuthEndpointsDocumentOAuthErrors(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/oauth/token", "/oauth/revoke"} {
		op := doc.Paths.Find(path).Post
		for _, status := range []int{400, 401} {
			res := op.Responses.Status(status)
			if res == nil || res.Value.Content.Get("application/json") == nil ||
				!strings.HasSuffix(res.Value.Content.Get("application/json").Schema.Ref, "/OAuthError") {
				t.Errorf("%s %d is not an OAuthError", path, status)
			}
		}
	}
}

// The reasons the OAuth error page is given are the ones the document
// lists and the ones the web app explains, no more and no fewer.
func TestOAuthErrorReasonsAreDocumentedAndExplained(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	want := slices.Sorted(slices.Values(app.Reasons()))
	schema := doc.Components.Schemas["OAuthErrorReason"]
	if schema == nil {
		t.Fatal("the document has no OAuthErrorReason")
	}
	var documented []string
	for _, v := range schema.Value.Enum {
		documented = append(documented, fmt.Sprint(v))
	}
	if slices.Sort(documented); !slices.Equal(documented, want) {
		t.Errorf("the document lists %v, want %v", documented, want)
	}
	for _, reason := range want {
		if !strings.Contains(schema.Value.Description, "- "+reason+": ") {
			t.Errorf("the document doesn't say what %s means", reason)
		}
	}

	screen, err := os.ReadFile(oauthErrorScreen)
	if err != nil {
		t.Fatal(err)
	}
	var explained []string
	for _, m := range webReason.FindAllStringSubmatch(string(screen), -1) {
		if m[1] != m[2] {
			t.Errorf("the web app explains %s with the message of %s", m[1], m[2])
		}
		explained = append(explained, m[1])
	}
	if slices.Sort(explained); !slices.Equal(explained, want) {
		t.Errorf("the web app explains %v, want %v", explained, want)
	}
}

const oauthErrorScreen = "../../../web/src/screens/OAuthError.tsx"

// webReason is an entry of the web app's list of reasons, such as
// ['pkce_required', appFault('oauthError.pkce_required')].
var webReason = regexp.MustCompile(`\['([a-z_]+)', [a-zA-Z]+\('oauthError\.([a-z_]+)'\)\]`)
