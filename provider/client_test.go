package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/openai/openai-go/v3/shared"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/tool"
)

const toolCallReply = `{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[` +
	`{"id":"c9","type":"function","function":{"name":"look","arguments":"{\"a\":1}"}}]}}]}`

// serve returns a client talking to a fake server that replies with reply, and the body it received.
func serve(t *testing.T, reply string) (*Client, *map[string]any) {
	t.Helper()

	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Error(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	return New(server.URL, "key"), &got
}

func TestSendWithoutTools(t *testing.T) {
	client, got := serve(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)

	if _, err := client.Send(context.Background(), agent.Agent{}, []Message{User("x")}); err != nil {
		t.Fatal(err)
	}

	if _, ok := (*got)["tools"]; ok {
		t.Error("tools sent while nil")
	}

	for _, key := range []string{"reasoning_effort", "max_completion_tokens"} {
		if _, ok := (*got)[key]; ok {
			t.Errorf("%s sent while zero", key)
		}
	}
}

func TestSendAgent(t *testing.T) {
	client, got := serve(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)
	a := agent.New("m", "be brief", nil, agent.Config{Effort: shared.ReasoningEffortMedium, MaxTokens: 100})

	if _, err := client.Send(context.Background(), a, []Message{User("x")}); err != nil {
		t.Fatal(err)
	}

	messages := (*got)["messages"].([]any)
	first := messages[0].(map[string]any)
	second := messages[1].(map[string]any)
	if (*got)["model"] != "m" || len(messages) != 2 || first["role"] != "system" || first["content"] != "be brief" ||
		second["role"] != "user" || second["content"] != "x" {
		t.Errorf("model = %v, messages = %+v", (*got)["model"], messages)
	}

	if (*got)["reasoning_effort"] != "medium" || (*got)["max_completion_tokens"] != float64(100) {
		t.Errorf("config = %v, %v", (*got)["reasoning_effort"], (*got)["max_completion_tokens"])
	}
}

func TestSendTools(t *testing.T) {
	client, got := serve(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)
	schemas := []tool.Schema{tool.New("look", "d", &jsonschema.Schema{Type: "object"})}

	if _, err := client.Send(context.Background(), agent.Agent{Tools: schemas}, nil); err != nil {
		t.Fatal(err)
	}

	sent := (*got)["tools"].([]any)[0].(map[string]any)
	function := sent["function"].(map[string]any)
	parameters := function["parameters"].(map[string]any)
	if sent["type"] != "function" || function["name"] != "look" || function["description"] != "d" ||
		parameters["type"] != "object" {
		t.Errorf("tool = %+v", sent)
	}
}

func TestSendBadSchema(t *testing.T) {
	client, _ := serve(t, `{"choices":[]}`)
	schemas := []tool.Schema{tool.New("look", "d", &jsonschema.Schema{Type: "object", Types: []string{"object"}})}

	if _, err := client.Send(context.Background(), agent.Agent{Tools: schemas}, nil); err == nil {
		t.Error("want an error")
	}
}

func TestSendReply(t *testing.T) {
	client, _ := serve(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)

	reply, err := client.Send(context.Background(), agent.Agent{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if reply.Content != "hi" {
		t.Errorf("reply = %+v", reply)
	}

	client, _ = serve(t, toolCallReply)

	reply, err = client.Send(context.Background(), agent.Agent{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(reply.ToolCalls) != 1 {
		t.Fatalf("reply = %+v", reply)
	}

	call := reply.ToolCalls[0]
	if call.ID != "c9" || call.Function.Name != "look" || call.Function.Arguments != `{"a":1}` {
		t.Errorf("call = %+v", call)
	}
}

func TestSendNoChoices(t *testing.T) {
	client, _ := serve(t, `{"choices":[]}`)

	if _, err := client.Send(context.Background(), agent.Agent{}, nil); err == nil {
		t.Error("want an error")
	}
}
