package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/sparkforge/internal/router"
)

// Kano 1 (Security): API authentication must be 100% correct — every bypass is a vulnerability.

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	r, err := router.New()
	if err != nil {
		t.Fatalf("router.New() error: %v", err)
	}

	srv, err := New(r)
	if err != nil {
		t.Fatalf("api.New() error: %v", err)
	}
	return srv, home
}

func addTestToken(t *testing.T, home, name, token string) {
	t.Helper()
	secretsPath := home + "/.forge/secrets.age"
	sec, err := secrets.New(secretsPath)
	if err != nil {
		t.Fatalf("secrets.New() error: %v", err)
	}
	if err := sec.Set("sparkforge.api_tokens."+name, token); err != nil {
		t.Fatalf("secrets.Set() error: %v", err)
	}
}

func doNotify(t *testing.T, srv *Server, body string, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/notify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify", srv.handleNotify)
	mux.HandleFunc("GET /health", srv.handleHealth)
	mux.ServeHTTP(w, req)
	return w
}

// --- Auth tests (Kano 1: Security) ---

func TestAPI_NoToken_Returns401(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doNotify(t, srv, `{"title":"test","priority":"low","source":"x"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no auth: status = %d, want 401", w.Code)
	}
}

func TestAPI_WrongToken_Returns401(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "good", "real-token-abc")
	w := doNotify(t, srv, `{"title":"test","priority":"low","source":"x"}`, "Bearer wrong-token")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: status = %d, want 401", w.Code)
	}
}

func TestAPI_ValidToken_Returns200(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "ci-bot", "secret-ci-token")
	w := doNotify(t, srv, `{"title":"Deploy done","priority":"medium","source":"smeltforge"}`, "Bearer secret-ci-token")
	if w.Code != http.StatusOK {
		t.Errorf("valid token: status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestAPI_BearerPrefixRequired(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "tok", "abc123")

	// Raw token without Bearer prefix should fail.
	w := doNotify(t, srv, `{"title":"t","priority":"low","source":"x"}`, "abc123")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("token without Bearer prefix: status = %d, want 401", w.Code)
	}

	// Token= prefix (wrong scheme) should fail.
	w = doNotify(t, srv, `{"title":"t","priority":"low","source":"x"}`, "Token abc123")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Token scheme: status = %d, want 401", w.Code)
	}

	// Empty Bearer should fail.
	w = doNotify(t, srv, `{"title":"t","priority":"low","source":"x"}`, "Bearer ")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("empty Bearer: status = %d, want 401", w.Code)
	}
}

func TestAPI_TokenFromDifferentNamespace_Rejected(t *testing.T) {
	srv, home := newTestServer(t)
	// Store a secret under a different prefix — must NOT be usable as an API token.
	secretsPath := home + "/.forge/secrets.age"
	sec, _ := secrets.New(secretsPath)
	_ = sec.Set("other.namespace.token", "injected-token")

	w := doNotify(t, srv, `{"title":"t","priority":"low","source":"x"}`, "Bearer injected-token")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("cross-namespace token: status = %d, want 401", w.Code)
	}
}

// --- Input validation tests ---

func TestAPI_MissingTitle_Returns400(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	w := doNotify(t, srv, `{"priority":"low","source":"x"}`, "Bearer tok")
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing title: status = %d, want 400", w.Code)
	}
}

func TestAPI_MissingSource_Returns400(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	w := doNotify(t, srv, `{"title":"t","priority":"low"}`, "Bearer tok")
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing source: status = %d, want 400", w.Code)
	}
}

func TestAPI_InvalidPriority_Returns400(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	for _, bad := range []string{"urgent", "HIGH", "5", ""} {
		body := `{"title":"t","priority":"` + bad + `","source":"x"}`
		w := doNotify(t, srv, body, "Bearer tok")
		if w.Code != http.StatusBadRequest {
			t.Errorf("priority=%q: status = %d, want 400", bad, w.Code)
		}
	}
}

func TestAPI_InvalidJSON_Returns400(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	w := doNotify(t, srv, `not-json`, "Bearer tok")
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON: status = %d, want 400", w.Code)
	}
}

// --- Response shape tests ---

func TestAPI_ResponseContainsChannelsList(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	w := doNotify(t, srv, `{"title":"t","priority":"low","source":"test"}`, "Bearer tok")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["status"]; !ok {
		t.Error("response missing 'status' field")
	}
	if _, ok := resp["channels"]; !ok {
		t.Error("response missing 'channels' field")
	}
}

func TestAPI_Health_Returns200(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", srv.handleHealth)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("health: status = %d, want 200", w.Code)
	}
}

// --- Payload field tests ---

func TestAPI_AllPriorityLevelsAccepted(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	for _, p := range []string{"low", "medium", "high", "critical"} {
		body, _ := json.Marshal(map[string]string{
			"title": "test", "priority": p, "source": "test",
		})
		w := doNotify(t, srv, string(body), "Bearer tok")
		if w.Code != http.StatusOK {
			t.Errorf("priority=%q: status = %d, want 200", p, w.Code)
		}
	}
}

func TestAPI_OptionalBodyAndLink(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	payload := map[string]string{
		"title":    "Backup done",
		"body":     "All 42 files backed up in 3.2s",
		"priority": "low",
		"source":   "backup-cron",
		"link":     "https://logs.example.com/backup/123",
	}
	b, _ := json.Marshal(payload)
	w := doNotify(t, srv, string(b), "Bearer tok")
	if w.Code != http.StatusOK {
		t.Errorf("with body+link: status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestAPI_EventType_FieldAccepted(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	payload := map[string]string{
		"title":      "monitor down",
		"priority":   "high",
		"source":     "watchforge",
		"event_type": "monitor:api-health",
	}
	b, _ := json.Marshal(payload)
	w := doNotify(t, srv, string(b), "Bearer tok")
	if w.Code != http.StatusOK {
		t.Errorf("with event_type: status = %d, want 200", w.Code)
	}
}

// Ensure JSON body is parsed correctly even with extra whitespace.
func TestAPI_ExtraWhitespaceInBody(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")
	body := `  {  "title"  :  "test"  ,  "priority"  :  "low"  ,  "source"  :  "x"  }  `
	w := doNotify(t, srv, body, "Bearer tok")
	if w.Code != http.StatusOK {
		t.Errorf("whitespace body: status = %d, want 200", w.Code)
	}
}

// POST to non-existent route returns 404 (method routing works).
func TestAPI_UnknownRoute_Returns405Or404(t *testing.T) {
	srv, home := newTestServer(t)
	addTestToken(t, home, "t", "tok")

	req := httptest.NewRequest("GET", "/notify", bytes.NewReader(nil))
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify", srv.handleNotify)
	mux.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Errorf("GET /notify should not return 200")
	}
}
