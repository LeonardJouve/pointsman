package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

// serve returns a handler whose model answers every request with the given status and body.
func serve(t *testing.T, code int, body string) http.Handler {
	t.Helper()

	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write([]byte(body))
	}))
	t.Cleanup(model.Close)

	return New(provider.New(model.URL, "key"), nil).Handler()
}

func send(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))

	return recorder
}

// start creates an agent and a session on it, and returns the session path.
func start(t *testing.T, handler http.Handler) string {
	t.Helper()

	var agent, session idResponse
	created := send(handler, "POST", "/agents", `{"model":"m"}`)
	if err := json.Unmarshal(created.Body.Bytes(), &agent); err != nil {
		t.Fatal(err)
	}

	request, _ := json.Marshal(sessionRequest{Agent: agent.ID.String(), Prompt: "hello"})
	created = send(handler, "POST", "/sessions", string(request))
	if err := json.Unmarshal(created.Body.Bytes(), &session); err != nil || created.Code != http.StatusCreated {
		t.Fatalf("got %d %s", created.Code, created.Body)
	}

	return "/sessions/" + session.ID.String()
}

// finished polls the session until it is no longer running.
func finished(t *testing.T, handler http.Handler, path string) (status session.Status, messages []provider.Message) {
	t.Helper()

	for range 200 {
		var got sessionResponse
		if err := json.Unmarshal(send(handler, "GET", path, "").Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}

		if got.Status != "running" {
			return got.Status, got.Messages
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("session still running")

	return "", nil
}

func TestSessionSucceeds(t *testing.T) {
	handler := serve(t, 200, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)
	path := start(t, handler)

	status, messages := finished(t, handler, path)
	if status != "succeeded" || len(messages) != 2 ||
		messages[0].OfUser == nil || messages[0].OfUser.Content.OfString.Value != "hello" ||
		messages[1].OfAssistant == nil || messages[1].OfAssistant.Content.OfString.Value != "done" {
		t.Errorf("got %s %v", status, messages)
	}
}

func TestSessionFails(t *testing.T) {
	handler := serve(t, 500, `{}`)
	path := start(t, handler)

	if status, _ := finished(t, handler, path); status != "failed" {
		t.Errorf("got %s", status)
	}
}

func TestCreateSessionErrors(t *testing.T) {
	handler := New(nil, nil).Handler()

	for name, body := range map[string]string{
		"bad json":      `{`,
		"bad agent id":  `{"agent":"nope"}`,
		"unknown agent": `{"agent":"6ba7b810-9dad-11d1-80b4-00c04fd430c8"}`,
	} {
		if code := send(handler, "POST", "/sessions", body).Code; code != http.StatusBadRequest {
			t.Errorf("%s: got %d", name, code)
		}
	}
}

func TestGetUnknownSession(t *testing.T) {
	handler := New(nil, nil).Handler()

	for _, id := range []string{"nope", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
		if code := send(handler, "GET", "/sessions/"+id, "").Code; code != http.StatusNotFound {
			t.Errorf("%s: got %d", id, code)
		}
	}
}
