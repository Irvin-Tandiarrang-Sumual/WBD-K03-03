package docs_test

import (
	"encoding/json"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/docs"
)

func TestSwaggerInfo_ReadDoc_ReturnsValidJSONWithDraftPaths(t *testing.T) {
	// Arrange & Act
	rendered := docs.SwaggerInfo.ReadDoc()

	// Assert
	var parsed map[string]any
	if err := json.Unmarshal([]byte(rendered), &parsed); err != nil {
		t.Fatalf("expected SwaggerInfo.ReadDoc() to return valid JSON, got error: %v", err)
	}

	schemes, ok := parsed["schemes"].([]any)
	if !ok || len(schemes) == 0 {
		t.Fatalf("expected schemes to be a non-empty array of strings, got %T (%v)", parsed["schemes"], parsed["schemes"])
	}
	if _, isStr := schemes[0].(string); !isStr {
		t.Fatalf("expected schemes element to be string, got %T", schemes[0])
	}

	if parsed["basePath"] != "/api/v1" {
		t.Fatalf("expected basePath '/api/v1', got %v", parsed["basePath"])
	}
	if parsed["host"] != "localhost:8080" {
		t.Fatalf("expected host 'localhost:8080', got %v", parsed["host"])
	}

	paths, ok := parsed["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object, got %T", parsed["paths"])
	}

	expectedPaths := []string{
		"/databases/{id}/drafts",
		"/drafts/{id}",
		"/drafts/{id}/approvals",
	}
	for _, p := range expectedPaths {
		if _, exists := paths[p]; !exists {
			t.Fatalf("expected path %q to exist in rendered swagger spec", p)
		}
	}
}
