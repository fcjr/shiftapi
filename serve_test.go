//go:build shiftapidev

package shiftapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestExportSpec(t *testing.T) {
	api := New(WithInfo(Info{
		Title:   "Export Test",
		Version: "1.0.0",
	}))
	Get(api, "/health", func(r *http.Request, _ struct{}) (*struct {
		OK bool `json:"ok"`
	}, error) {
		return &struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	})

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.json")

	if err := exportSpec(api, specPath); err != nil {
		t.Fatalf("exportSpec failed: %v", err)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("spec is not valid JSON: %v", err)
	}

	if spec["openapi"] != "3.1" {
		t.Errorf("expected openapi 3.1, got %v", spec["openapi"])
	}

	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatal("expected info in spec")
	}
	if info["title"] != "Export Test" {
		t.Errorf("expected title 'Export Test', got %v", info["title"])
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths in spec")
	}
	if _, ok := paths["/health"]; !ok {
		t.Error("expected /health in paths")
	}
}

func TestExportSpecInvalidPath(t *testing.T) {
	api := New()
	err := exportSpec(api, "/nonexistent/dir/spec.json")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
