package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/store"
	"uuid"
)

func lines(t *testing.T, s *Session) []string {
	t.Helper()

	dir, err := store.Sessions()
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, s.ID.String()+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	return strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
}

func TestFile(t *testing.T) {
	isolate(t)

	a := &agent.Agent{ID: uuid.New()}
	s, err := New(a)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.AppendUser("hi"); err != nil {
		t.Fatal(err)
	}

	if err := s.SetStatus(Succeeded); err != nil {
		t.Fatal(err)
	}

	want := []string{
		`{"header":{"created":"` + s.Created.Format(time.RFC3339Nano) + `","agent":"` + a.ID.String() + `"}}`,
		`{"message":{"content":"hi","role":"user"}}`,
		`{"status":"succeeded"}`,
	}

	got := lines(t, s)
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %s, want %s", i, got[i], want[i])
		}
	}
}

func TestCreateExists(t *testing.T) {
	s := start(t)

	if err := create(s); !os.IsExist(err) {
		t.Errorf("got %v, want file exists", err)
	}
}

func TestWriteFails(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AppData", blocker)
	t.Setenv("XDG_CONFIG_HOME", blocker)

	if _, err := New(&agent.Agent{}); err == nil {
		t.Error("New: no error")
	}

	s := &Session{ID: uuid.New(), status: Running, changed: make(chan struct{})}

	if err := s.AppendUser("hi"); err == nil || len(s.Messages()) != 0 {
		t.Errorf("AppendUser: err %v, %d messages", err, len(s.Messages()))
	}

	changed := s.Changed()
	if err := s.SetStatus(Failed); err == nil || s.Status() != Failed {
		t.Errorf("SetStatus: err %v, status %v", err, s.Status())
	}
	<-changed
}

func TestAppendRoundFails(t *testing.T) {
	s := start(t)

	var reply provider.Reply
	if err := json.Unmarshal([]byte(twoCallsReply), &reply); err != nil {
		t.Fatal(err)
	}

	// A directory in place of the file makes the open fail.
	dir, err := store.Sessions()
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, s.ID.String()+".jsonl")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}

	changed := s.Changed()
	if err := s.AppendRound(reply, []string{"one", "two"}); err == nil || len(s.Messages()) != 0 {
		t.Errorf("err %v, %d messages", err, len(s.Messages()))
	}

	select {
	case <-changed:
		t.Error("notified after a failed write")
	default:
	}
}

func marshal(t *testing.T, messages []provider.Message) []string {
	t.Helper()

	var got []string
	for _, m := range messages {
		content, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}

		got = append(got, string(content))
	}

	return got
}

func TestLoadRoundTrip(t *testing.T) {
	isolate(t)

	a := &agent.Agent{ID: uuid.New()}
	s, err := New(a)
	if err != nil {
		t.Fatal(err)
	}

	var withCalls, plain provider.Reply
	if err := json.Unmarshal([]byte(twoCallsReply), &withCalls); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal([]byte(`{"role":"assistant","content":"done"}`), &plain); err != nil {
		t.Fatal(err)
	}

	for _, err := range []error{
		s.AppendUser("fix the build"),
		s.AppendRound(withCalls, []string{"one", "two"}),
		s.AppendRound(plain, nil),
		s.SetStatus(Failed),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := Load(agent.Agents{a.ID: a})
	if err != nil {
		t.Fatal(err)
	}

	got := sessions[s.ID]
	if got == nil || got.Agent != a || !got.Created.Equal(s.Created) || got.Status() != Failed || got.changed == nil {
		t.Fatalf("got %+v", got)
	}

	want, have := marshal(t, s.Messages()), marshal(t, got.Messages())
	if len(want) != 5 || !slices.Equal(want, have) {
		t.Errorf("got %q, want %q", have, want)
	}
}

func TestLoadStatus(t *testing.T) {
	s := start(t)
	a := agent.Agents{{}: s.Agent}

	sessions, err := Load(a)
	if err != nil || sessions[s.ID].Status() != Running {
		t.Errorf("no status line: err %v, got %+v", err, sessions[s.ID])
	}

	if err := s.SetStatus(Succeeded); err != nil {
		t.Fatal(err)
	}

	sessions, err = Load(a)
	if err != nil || sessions[s.ID].Status() != Succeeded {
		t.Errorf("status line: err %v, got %+v", err, sessions[s.ID])
	}
}

func TestLoadEmpty(t *testing.T) {
	isolate(t)

	sessions, err := Load(nil)
	if err != nil || len(sessions) != 0 {
		t.Errorf("got %v, %v", sessions, err)
	}
}

func TestLoadFails(t *testing.T) {
	known := uuid.New()
	header := `{"header":{"created":"2026-01-02T03:04:05Z","agent":"` + known.String() + `"}}`
	agents := agent.Agents{known: {ID: known}}

	for name, content := range map[string]string{
		"not a header":  `{"status":"failed"}` + "\n",
		"bad json":      header + "\n{\n",
		"unknown kind":  header + "\n" + `{"note":"x"}` + "\n",
		"unknown agent": `{"header":{"created":"2026-01-02T03:04:05Z","agent":"` + uuid.New().String() + `"}}` + "\n",
	} {
		isolate(t)

		dir, err := store.Sessions()
		if err != nil {
			t.Fatal(err)
		}

		file := filepath.Join(dir, uuid.New().String()+".jsonl")
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(agents); err == nil || !strings.Contains(err.Error(), file) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
