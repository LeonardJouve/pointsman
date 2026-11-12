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
	created := send(handler, "POST", "/agents", `{"name":"a","model":"m"}`)
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

// events reads a stream request to its end and returns the body.
func events(handler http.Handler, path string) (code int, body string) {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", path+"/events", nil))

	return recorder.Code, recorder.Body.String()
}

func TestStreamFinishedSession(t *testing.T) {
	handler := serve(t, 200, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)
	path := start(t, handler)
	finished(t, handler, path)

	code, body := events(handler, path)
	want := "id: 0\nevent: message\ndata: {\"content\":\"hello\",\"role\":\"user\"}\n\n" +
		"id: 1\nevent: message\ndata: {\"content\":\"done\",\"role\":\"assistant\"}\n\n" +
		"event: status\ndata: {\"status\":\"succeeded\"}\n\n"
	if code != http.StatusOK || body != want {
		t.Errorf("got %d %q, want %q", code, body, want)
	}
}

func TestStreamRunningSession(t *testing.T) {
	release := make(chan struct{})
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"done"}}]}`))
	}))
	t.Cleanup(model.Close)

	handler := New(provider.New(model.URL, "key"), nil).Handler()
	path := start(t, handler)
	done := make(chan string)

	go func() {
		_, body := events(handler, path)
		done <- body
	}()

	time.Sleep(50 * time.Millisecond)
	close(release)

	body := <-done
	if !strings.Contains(body, "id: 1\nevent: message\ndata: {\"content\":\"done\"") ||
		!strings.HasSuffix(body, "event: status\ndata: {\"status\":\"succeeded\"}\n\n") {
		t.Errorf("got %q", body)
	}
}

func TestStreamUnknownSession(t *testing.T) {
	handler := New(nil, nil).Handler()

	for _, id := range []string{"nope", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
		if code, _ := events(handler, "/sessions/"+id); code != http.StatusNotFound {
			t.Errorf("%s: got %d", id, code)
		}
	}
}

func TestStreamResume(t *testing.T) {
	handler := serve(t, 200, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)
	path := start(t, handler)
	finished(t, handler, path)

	status := "event: status\ndata: {\"status\":\"succeeded\"}\n\n"
	first := "id: 0\nevent: message\ndata: {\"content\":\"hello\",\"role\":\"user\"}\n\n"
	second := "id: 1\nevent: message\ndata: {\"content\":\"done\",\"role\":\"assistant\"}\n\n"

	for last, want := range map[string]string{
		"0":                   second + status,
		"1":                   status,
		"99":                  status,
		"-1":                  first + second + status,
		"9223372036854775807": first + second + status,
	} {
		request := httptest.NewRequest("GET", path+"/events", nil)
		request.Header.Set("Last-Event-ID", last)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		if recorder.Body.String() != want {
			t.Errorf("%s: got %q, want %q", last, recorder.Body.String(), want)
		}
	}
}

func TestWriteKeepAlive(t *testing.T) {
	var out strings.Builder
	writeKeepAlive(&out)

	if want := ": keep-alive\n\n"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestListSessions(t *testing.T) {
	handler := serve(t, 200, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)

	if got := strings.TrimSpace(send(handler, "GET", "/sessions", "").Body.String()); got != "{}" {
		t.Fatalf("got %s", got)
	}

	path := start(t, handler)
	finished(t, handler, path)

	var got map[string]sessionSummary
	if err := json.Unmarshal(send(handler, "GET", "/sessions", "").Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	listed, ok := got[strings.TrimPrefix(path, "/sessions/")]
	if !ok || listed.Created.IsZero() || listed.Status != session.Succeeded || listed.Agent.Name != "a" {
		t.Errorf("got %+v", got)
	}
}
