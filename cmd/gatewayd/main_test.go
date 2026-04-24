package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareUIAssetsUsesBundledDirectory(t *testing.T) {
	t.Parallel()

	uiDir := filepath.Join(t.TempDir(), "metacubexd")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll returned error: %v", err)
	}
	indexPath := filepath.Join(uiDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	resolved, err := prepareUIAssets(uiDir)
	if err != nil {
		t.Fatalf("prepareUIAssets returned error: %v", err)
	}
	if resolved != uiDir {
		t.Fatalf("resolved ui dir = %q, want %q", resolved, uiDir)
	}

	configJS, err := os.ReadFile(filepath.Join(uiDir, "config.js"))
	if err != nil {
		t.Fatalf("os.ReadFile returned error: %v", err)
	}
	if string(configJS) == "" {
		t.Fatal("config.js is empty")
	}
}

func TestPrepareUIAssetsRejectsMissingBundledIndex(t *testing.T) {
	t.Parallel()

	uiDir := t.TempDir()

	if _, err := prepareUIAssets(uiDir); err == nil {
		t.Fatal("prepareUIAssets returned nil error, want missing index error")
	}
}

func TestNewUIHandlerServesIndexAtRoot(t *testing.T) {
	t.Parallel()

	uiDir := filepath.Join(t.TempDir(), "metacubexd")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "config.js"), []byte("window.test = 1"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	handler, err := newUIHandler(uiDir, "http://127.0.0.1:9090", "")
	if err != nil {
		t.Fatalf("newUIHandler returned error: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "<html>ok</html>" {
		t.Fatalf("body = %q, want index html", got)
	}
}
