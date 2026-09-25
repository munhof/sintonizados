package httpadapter_test

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Compare registered literal routes with the derived contract in both directions.
// Runtime response schemas are checked by scripts/check-contract.py against the OCI server.
func TestPublicRoutesAreModeled(t *testing.T) {
	b, err := os.ReadFile("../../../api/generated/openapi/sintonizados.openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OpenAPI != "3.1.0" {
		t.Fatal(spec.OpenAPI)
	}
	source, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`mux.HandleFunc\("(GET|POST) ([^"]+)"`)
	seen := map[string]bool{}
	for _, m := range pattern.FindAllStringSubmatch(string(source), -1) {
		path := strings.ReplaceAll(m[2], "{$}", "")
		method := strings.ToLower(m[1])
		if _, ok := spec.Paths[path][method]; !ok {
			t.Errorf("unmodeled route %s %s", method, path)
		}
		seen[method+" "+path] = true
	}
	for path, methods := range spec.Paths {
		for method := range methods {
			if !seen[method+" "+path] {
				t.Errorf("modeled operation missing implementation: %s %s", method, path)
			}
		}
	}
}
