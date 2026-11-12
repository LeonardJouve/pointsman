package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
)

// isolate points the user config folder to a temporary one.
func isolate(t *testing.T, folder string) {
	t.Helper()
	t.Setenv("AppData", folder)
	t.Setenv("XDG_CONFIG_HOME", folder)
}

func newHandler(t *testing.T, client *provider.Client) http.Handler {
	t.Helper()
	isolate(t, t.TempDir())

	return New(client, nil, map[uuid.UUID]*agent.Agent{}).Handler()
}

func do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := newHandler(t, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))

	return recorder
}

func TestCreateAndListAgents(t *testing.T) {
	handler := newHandler(t, nil)
	const request = `{"name":"n","model":"m","prompt":"p","tools":["write"],"effort":"low","max_tokens":10}`

	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest("POST", "/agents", strings.NewReader(request)))

	var id idResponse
	if err := json.Unmarshal(created.Body.Bytes(), &id); err != nil || created.Code != http.StatusCreated {
		t.Fatalf("got %d %s", created.Code, created.Body)
	}

	got := httptest.NewRecorder()
	handler.ServeHTTP(got, httptest.NewRequest("GET", "/agents", nil))

	want := `{"` + id.ID.String() + `":{"id":"` + id.ID.String() + `",` + request[1:] + `}`
	if strings.TrimSpace(got.Body.String()) != want {
		t.Errorf("got %s, want %s", got.Body, want)
	}
}

func TestCreateAgentErrors(t *testing.T) {
	for name, body := range map[string]string{
		"unknown tool": `{"name":"n","model":"m","tools":["nope"]}`,
		"bad json":     `{`,
	} {
		if code := do(t, "POST", "/agents", body).Code; code != http.StatusBadRequest {
			t.Errorf("%s: got %d", name, code)
		}
	}
}

func TestListNoAgents(t *testing.T) {
	if got := strings.TrimSpace(do(t, "GET", "/agents", "").Body.String()); got != "{}" {
		t.Errorf("got %s", got)
	}
}

func TestDashboard(t *testing.T) {
	got := do(t, "GET", "/", "")
	if got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "<html") {
		t.Errorf("got %d %s", got.Code, got.Body)
	}
}

func TestCreateAgentSaves(t *testing.T) {
	folder := t.TempDir()
	isolate(t, folder)

	handler := New(nil, nil, map[uuid.UUID]*agent.Agent{}).Handler()
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest("POST", "/agents", strings.NewReader(`{"name":"n","model":"m"}`)))

	var id idResponse
	if err := json.Unmarshal(created.Body.Bytes(), &id); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(folder, "pointsman", "agents", id.ID.String()+".json")); err != nil {
		t.Error(err)
	}
}

func TestCreateAgentSaveFails(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	isolate(t, blocker)

	handler := New(nil, nil, map[uuid.UUID]*agent.Agent{}).Handler()
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest("POST", "/agents", strings.NewReader(`{"name":"n","model":"m"}`)))

	got := httptest.NewRecorder()
	handler.ServeHTTP(got, httptest.NewRequest("GET", "/agents", nil))

	if created.Code != http.StatusInternalServerError || strings.TrimSpace(got.Body.String()) != "{}" {
		t.Errorf("got %d, list %s", created.Code, got.Body)
	}
}
