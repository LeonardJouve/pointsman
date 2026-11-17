package tool

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LeonardJouve/pointsman/workspace"
)

// fakeRouter serves POST /sessions with createStatus and, when it is 201, the given events body.
func fakeRouter(t *testing.T, createStatus int, events string) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /sessions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(createStatus)
		if createStatus == http.StatusCreated {
			fmt.Fprint(w, `{"id":"child"}`)
		} else {
			fmt.Fprint(w, `{"error":"no such agent"}`)
		}
	})
	mux.HandleFunc("GET /sessions/child/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, events)
		w.(http.Flusher).Flush()

		if strings.HasSuffix(events, "\n\n") {
			return
		}

		<-r.Context().Done()
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Setenv("ROUTER_ADDR", strings.TrimPrefix(server.URL, "http://"))
}

const subagentArgs = `{"agent":"a","prompt":"p"}`

func TestSubagent(t *testing.T) {
	fakeRouter(t, http.StatusCreated, "id: 0\nevent: message\ndata: {\"role\":\"user\",\"content\":\"p\"}\n\n"+
		"id: 1\nevent: message\ndata: {\"role\":\"assistant\",\"content\":\"first\"}\n\n"+
		"id: 2\nevent: message\ndata: {\"role\":\"assistant\",\"content\":\"done\"}\n\n"+
		"event: status\ndata: {\"status\":\"succeeded\"}\n\n")

	output, code := subagent(context.Background(), workspace.Workspace{}, subagentArgs)
	if code != 0 || output != "done" {
		t.Errorf("got %q, %d", output, code)
	}
}

func TestSubagentFailed(t *testing.T) {
	fakeRouter(t, http.StatusCreated, "event: status\ndata: {\"status\":\"failed\"}\n\n")

	if _, code := subagent(context.Background(), workspace.Workspace{}, subagentArgs); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}

func TestSubagentCreateRefused(t *testing.T) {
	fakeRouter(t, http.StatusBadRequest, "")

	output, code := subagent(context.Background(), workspace.Workspace{}, subagentArgs)
	if code != 1 || !strings.Contains(output, "no such agent") {
		t.Errorf("got %q, %d", output, code)
	}
}

func TestSubagentRouterNotSet(t *testing.T) {
	t.Setenv("ROUTER_ADDR", "")

	if _, code := subagent(context.Background(), workspace.Workspace{}, subagentArgs); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}

func TestSubagentBadArguments(t *testing.T) {
	if _, code := subagent(context.Background(), workspace.Workspace{}, "{"); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}

func TestSubagentCancelled(t *testing.T) {
	fakeRouter(t, http.StatusCreated, "id: 0\nevent: message\ndata: {\"content\":\"p\"}\n\n")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if _, code := subagent(ctx, workspace.Workspace{}, subagentArgs); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}
