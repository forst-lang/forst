package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forst/internal/discovery"
)

type stubTypesGen struct {
	err error
}

func (s *stubTypesGen) GenerateTypesForFunctions(map[string]map[string]discovery.FunctionInfo, string) (string, error) {
	return "", s.err
}

func TestHandleTypes_generateTypesError_returns500(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = &stubTypesGen{err: fmt.Errorf("types")}
	s.typesCacheMu.Lock()
	s.typesCache["types"] = ""
	s.lastTypesGen = time.Now()
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types?force=true", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleTypes_get_returnsJSON_wrongMethod(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types: %d %s", rr.Code, rr.Body.String())
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil || !resp.Success {
		t.Fatalf("decode: %v resp=%+v", err, resp)
	}

	rr2 := httptest.NewRecorder()
	s.handleTypes(rr2, httptest.NewRequest(http.MethodPost, "/types", nil))
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("bad method: %d", rr2.Code)
	}
}

func TestHandleTypes_freshCache_returnsCachedWithoutGenerator(t *testing.T) {
	s := testDevServer(t)
	s.typesCacheMu.Lock()
	s.typesCache["types"] = "cached-types-content"
	s.lastTypesGen = time.Now()
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types: %d %s", rr.Code, rr.Body.String())
	}

	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.Output != "cached-types-content" {
		t.Fatalf("expected cached output, got %q", resp.Output)
	}
}

func TestHandleTypes_forceRegenerate_overwritesCache(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	s.typesCacheMu.Lock()
	s.typesCache["types"] = "stale-cache"
	s.lastTypesGen = time.Now()
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types?force=true", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types?force=true: %d %s", rr.Code, rr.Body.String())
	}

	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.Output == "stale-cache" {
		t.Fatalf("expected regenerated output, got stale cache")
	}
}

func TestHandleTypes_staleCache_regeneratesWithoutForce(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	s.typesCacheMu.Lock()
	s.typesCache["types"] = "stale-cache"
	s.lastTypesGen = time.Now().Add(-6 * time.Minute)
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types stale cache: %d %s", rr.Code, rr.Body.String())
	}

	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.Output == "stale-cache" {
		t.Fatalf("expected stale cache invalidation and regeneration")
	}
}

func TestHandleTypes_discoveryFailureOnRegenerate_returns500(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	s.discoverer = discovery.NewDiscoverer(t.TempDir(), s.log, nil)

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types?force=true", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}
