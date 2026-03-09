//go:build shiftapidev

package shiftapi

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDevListenerStartsOnFirstServeHTTP(t *testing.T) {
	api := New(WithInfo(Info{
		Title:   "Dev Test",
		Version: "1.0.0",
	}))
	Get(api, "/health", func(r *http.Request, _ struct{}) (*struct {
		OK bool `json:"ok"`
	}, error) {
		return &struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	})

	if api.devListener == nil {
		t.Fatal("expected devListener to be set in shiftapidev build")
	}

	devAddr := api.devListener.Addr().String()

	// Before any ServeHTTP call, the dev listener should NOT be serving yet.
	// Try a quick connection — it should connect (port is bound) but the
	// server won't respond because http.Serve hasn't been called.
	conn, err := net.DialTimeout("tcp", devAddr, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("expected dev port to be bound: %v", err)
	}
	conn.Close()

	// Start a listener for the "user's" server.
	userLn, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer userLn.Close()
	go http.Serve(userLn, api) // triggers ServeHTTP → sync.Once → devServe

	// Send a request to the user's listener to trigger ServeHTTP.
	userURL := "http://" + userLn.Addr().String() + "/health"
	resp, err := http.Get(userURL)
	if err != nil {
		t.Fatalf("request to user listener failed: %v", err)
	}
	resp.Body.Close()

	// Now the dev listener should be serving. Fetch the spec from it.
	devURL := "http://" + devAddr + "/openapi.json"
	resp, err = http.Get(devURL)
	if err != nil {
		t.Fatalf("request to dev listener failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from dev listener, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("spec is not valid JSON: %v", err)
	}
	if spec["openapi"] != "3.1" {
		t.Errorf("expected openapi 3.1, got %v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths in spec")
	}
	if _, ok := paths["/health"]; !ok {
		t.Error("expected /health in paths")
	}
}
