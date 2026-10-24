package httpapi_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
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
