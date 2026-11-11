package session

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
)

const toolCallReply = `{"role":"assistant","content":null,"tool_calls":[` +
	`{"id":"c9","type":"function","function":{"name":"look","arguments":"{}"}}]}`

func TestNew(t *testing.T) {
	s := New(agent.Agent{})

	if s.ID.Compare(New(agent.Agent{}).ID) == 0 || len(s.Messages()) != 0 {
		t.Errorf("got ID %v with %d messages", s.ID, len(s.Messages()))
	}
}

func TestStatus(t *testing.T) {
	s := New(agent.Agent{})
	if s.Status() != Running {
		t.Errorf("new session is %v, want Running", s.Status())
	}

	for _, want := range []Status{Succeeded, Failed} {
		s.SetStatus(want)
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

	s := New(agent.Agent{})
	s.AppendUser("fix the build")
	s.AppendReply(reply)
	s.AppendToolResult("c9", "done")

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
	s := New(agent.Agent{})

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			s.AppendUser("go")
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
