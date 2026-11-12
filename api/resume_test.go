package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

// unfinished builds a session through the real writers, with its prompt and no status line.
func unfinished(t *testing.T, a *agent.Agent) *session.Session {
	t.Helper()

	created, err := session.New(a)
	if err != nil {
		t.Fatal(err)
	}

	if err := created.AppendUser("hello"); err != nil {
		t.Fatal(err)
	}

	return created
}

func TestResume(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"done"}}]}`))
	}))
	t.Cleanup(model.Close)

	isolate(t, t.TempDir())

	built := agent.New("a", "m", "", nil, agent.Config{})
	a := &built
	running := unfinished(t, a)
	succeeded := unfinished(t, a)
	failed := unfinished(t, a)
	succeeded.SetStatus(session.Succeeded)
	failed.SetStatus(session.Failed)

	sessions := session.Sessions{running.ID: running, succeeded.ID: succeeded, failed.ID: failed}
	New(provider.New(model.URL, "key"), nil, agent.Agents{a.ID: a}, sessions).Resume()

	deadline := time.Now().Add(2 * time.Second)
	for {
		changed := running.Changed()
		if running.Status() != session.Running {
			break
		}

		select {
		case <-changed:
		case <-time.After(time.Until(deadline)):
			t.Fatal("session still running")
		}
	}

	if got := running.Status(); got != session.Succeeded {
		t.Errorf("got %s", got)
	}

	if messages := running.Messages(); len(messages) != 2 || messages[1].OfAssistant == nil {
		t.Errorf("got %v", messages)
	}

	loaded, err := session.Load(agent.Agents{a.ID: a})
	if err != nil {
		t.Fatal(err)
	}

	if got := loaded[running.ID]; got.Status() != session.Succeeded || len(got.Messages()) != 2 {
		t.Errorf("file: got %s %v", got.Status(), got.Messages())
	}

	if len(succeeded.Messages()) != 1 || succeeded.Status() != session.Succeeded {
		t.Errorf("succeeded changed: %s %v", succeeded.Status(), succeeded.Messages())
	}

	if len(failed.Messages()) != 1 || failed.Status() != session.Failed {
		t.Errorf("failed changed: %s %v", failed.Status(), failed.Messages())
	}
}
