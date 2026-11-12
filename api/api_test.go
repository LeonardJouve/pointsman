package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := New(nil, nil).Handler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))

	return recorder
}

func TestCreateAndListAgents(t *testing.T) {
	handler := New(nil, nil).Handler()
	const request = `{"model":"m","prompt":"p","tools":["write"],"effort":"low","max_tokens":10}`

	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest("POST", "/agents", strings.NewReader(request)))

	var id idResponse
	if err := json.Unmarshal(created.Body.Bytes(), &id); err != nil || created.Code != http.StatusCreated {
		t.Fatalf("got %d %s", created.Code, created.Body)
	}

	got := httptest.NewRecorder()
	handler.ServeHTTP(got, httptest.NewRequest("GET", "/agents", nil))

	want := `{"` + id.ID.String() + `":{"model":"m","prompt":"p","tools":["write"],"effort":"low","max_tokens":10}}`
	if strings.TrimSpace(got.Body.String()) != want {
		t.Errorf("got %s, want %s", got.Body, want)
	}
}

func TestCreateAgentErrors(t *testing.T) {
	for name, body := range map[string]string{
		"unknown tool": `{"model":"m","tools":["nope"]}`,
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
