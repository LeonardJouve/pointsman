package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/store"
)

const toolCallReply = `{"role":"assistant","content":null,"tool_calls":[` +
	`{"id":"c9","type":"function","function":{"name":"look","arguments":"{}"}}]}`

const twoCallsReply = `{"role":"assistant","content":null,"tool_calls":[` +
	`{"id":"a","type":"function","function":{"name":"look","arguments":"{}"}},` +
	`{"id":"b","type":"function","function":{"name":"look","arguments":"{}"}}]}`

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

func start(t *testing.T) *Session {
	t.Helper()

	s, err := New(&agent.Agent{})
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func TestNew(t *testing.T) {
	s := start(t)

	other, err := New(&agent.Agent{})
	if err != nil {
		t.Fatal(err)
	}

	if s.ID.Compare(other.ID) == 0 || len(s.Messages()) != 0 {
		t.Errorf("got ID %v with %d messages", s.ID, len(s.Messages()))
	}
}

func TestStatus(t *testing.T) {
	s := start(t)
	if s.Status() != Running {
		t.Errorf("new session is %v, want Running", s.Status())
	}

	for _, want := range []Status{Succeeded, Failed} {
		if err := s.SetStatus(want); err != nil {
			t.Fatal(err)
		}

		if got := s.Status(); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestAppend(t *testing.T) {
	var reply provider.Reply
	if err := json.Unmarshal([]byte(toolCallReply), &reply); err != nil {
		t.Fatal(err)
	}

	s := start(t)
	for _, err := range []error{s.AppendUser("fix the build"), s.AppendRound(reply, []string{"done"})} {
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(s.Messages()) != 3 {
		t.Fatalf("got %d messages, want 3", len(s.Messages()))
	}

	roles := []string{"user", "assistant", "tool"}
	for i, message := range s.Messages() {
		data, err := json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}

		var got struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				ID string `json:"id"`
			} `json:"tool_calls"`
		}
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}

		if got.Role != roles[i] {
			t.Errorf("message %d: role %q, want %q", i, got.Role, roles[i])
		}
		if i == 1 && (len(got.ToolCalls) != 1 || got.ToolCalls[0].ID != "c9") {
			t.Errorf("assistant message lost its tool call: %s", data)
		}
	}
}

func TestMessagesWhileAppending(t *testing.T) {
	s := start(t)

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			if err := s.AppendUser("go"); err != nil {
				t.Error(err)
			}
		}
	})
	for range 100 {
		s.Messages()
	}
	wg.Wait()

	if got := len(s.Messages()); got != 100 {
		t.Errorf("got %d messages, want 100", got)
	}
}

func TestChanged(t *testing.T) {
	s := start(t)
	changed := s.Changed()

	select {
	case <-changed:
		t.Fatal("closed before any change")
	default:
	}

	if err := s.AppendUser("a"); err != nil {
		t.Fatal(err)
	}
	<-changed

	changed = s.Changed()
	if err := s.SetStatus(Succeeded); err != nil {
		t.Fatal(err)
	}
	<-changed
}

func TestChangedWakesReader(t *testing.T) {
	s := start(t)
	changed := s.Changed()
	woken := make(chan struct{})

	go func() {
		<-changed
		close(woken)
	}()

	if err := s.AppendUser("a"); err != nil {
		t.Fatal(err)
	}
	<-woken
}

func TestAppendRoundTwoCalls(t *testing.T) {
	var reply provider.Reply
	if err := json.Unmarshal([]byte(twoCallsReply), &reply); err != nil {
		t.Fatal(err)
	}

	s := start(t)
	changed := s.Changed()

	if err := s.AppendRound(reply, []string{"one", "two"}); err != nil {
		t.Fatal(err)
	}

	<-changed

	if s.Changed() == changed {
		t.Error("Changed returned the closed channel")
	}

	want := []string{
		`{"message":{"tool_calls":[` +
			`{"id":"a","function":{"arguments":"{}","name":"look"},"type":"function"},` +
			`{"id":"b","function":{"arguments":"{}","name":"look"},"type":"function"}],"role":"assistant"}}`,
		`{"message":{"content":"one","tool_call_id":"a","role":"tool"}}`,
		`{"message":{"content":"two","tool_call_id":"b","role":"tool"}}`,
	}

	got := lines(t, s)[1:]
	if len(got) != len(want) || len(s.Messages()) != len(want) {
		t.Fatalf("file %q, %d messages in memory", got, len(s.Messages()))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %s, want %s", i, got[i], want[i])
		}

		memory, err := json.Marshal(map[string]any{"message": s.Messages()[i]})
		if err != nil {
			t.Fatal(err)
		}

		if string(memory) != want[i] {
			t.Errorf("message %d in memory: got %s, want %s", i, memory, want[i])
		}
	}
}

func TestAppendRoundNoToolCalls(t *testing.T) {
	var reply provider.Reply
	if err := json.Unmarshal([]byte(`{"role":"assistant","content":"done"}`), &reply); err != nil {
		t.Fatal(err)
	}

	s := start(t)
	if err := s.AppendRound(reply, nil); err != nil {
		t.Fatal(err)
	}

	if got := lines(t, s)[1:]; len(got) != 1 || len(s.Messages()) != 1 {
		t.Errorf("file %q, %d messages in memory", got, len(s.Messages()))
	}
}
