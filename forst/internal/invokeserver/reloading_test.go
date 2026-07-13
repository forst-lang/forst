package invokeserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleHealth_reportsReloadingMarker(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORST_BOUNDARY_ROOT", dir)
	markerPath := filepath.Join(dir, ".forst", "reloading")
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath, []byte(`{"reloading":true,"generation":3}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var resp Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || !resp.Reloading || resp.Generation != 3 {
		t.Fatalf("response: %+v", resp)
	}
}

func TestHandleInvoke_returns503WhenReloadMarkerSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORST_BOUNDARY_ROOT", dir)
	markerPath := filepath.Join(dir, ".forst", "reloading")
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath, []byte(`{"reloading":true,"generation":5}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestServer(t, &stubBackend{})
	body := `{"package":"main","function":"Echo","args":{}}`
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After: %q", got)
	}
	var resp Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Success || resp.Error != "reloading" || !resp.Reloading || resp.Generation != 5 {
		t.Fatalf("response: %+v", resp)
	}
}

func TestReadReloadMarker_missingFile(t *testing.T) {
	reloading, generation := ReadReloadMarker(t.TempDir())
	if reloading || generation != 0 {
		t.Fatalf("got reloading=%v generation=%d", reloading, generation)
	}
}
