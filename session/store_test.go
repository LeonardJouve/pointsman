package session

import (
	"encoding/json"
	"os"
	"path/filepath"
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
