package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestCompleteRequest(t *testing.T) {
	client, got := serve(t, toolCallReply)
	tools := []Tool{NewTool("look", "d", map[string]any{"type": "object"})}

	reply, err := client.Complete(context.Background(), "m", tools, []Message{System("be brief"), User("hello")})
	if err != nil {
		t.Fatal(err)
	}

	messages := []Message{System("be brief"), User("hello"), reply.ToParam(), ToolResult("c9", "done")}
	if _, err := client.Complete(context.Background(), "m", tools, messages); err != nil {
		t.Fatal(err)
	}

	if (*got)["model"] != "m" {
		t.Errorf("model = %v", (*got)["model"])
	}

	sent := (*got)["messages"].([]any)
	roles := []string{"system", "user", "assistant", "tool"}
	if len(sent) != len(roles) {
		t.Fatalf("messages = %v", sent)
	}

	for i, role := range roles {
		if sent[i].(map[string]any)["role"] != role {
			t.Errorf("message %d = %v, want role %s", i, sent[i], role)
		}
	}

	if sent[2].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["id"] != "c9" {
		t.Errorf("assistant message = %v", sent[2])
	}

	if sent[3].(map[string]any)["tool_call_id"] != "c9" {
		t.Errorf("tool message = %v", sent[3])
	}

	function := (*got)["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if function["name"] != "look" || function["description"] != "d" {
		t.Errorf("function = %v", function)
	}

	if function["parameters"].(map[string]any)["type"] != "object" {
		t.Errorf("parameters = %v", function["parameters"])
	}
}

func TestCompleteWithoutTools(t *testing.T) {
	client, got := serve(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)

	if _, err := client.Complete(context.Background(), "m", nil, []Message{User("x")}); err != nil {
		t.Fatal(err)
	}

	if _, ok := (*got)["tools"]; ok {
		t.Error("tools sent while nil")
	}
}

func TestCompleteReply(t *testing.T) {
	client, _ := serve(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)

	reply, err := client.Complete(context.Background(), "m", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if reply.Content != "hi" {
		t.Errorf("reply = %+v", reply)
	}

	client, _ = serve(t, toolCallReply)

	reply, err = client.Complete(context.Background(), "m", nil, nil)
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

func TestCompleteNoChoices(t *testing.T) {
	client, _ := serve(t, `{"choices":[]}`)

	if _, err := client.Complete(context.Background(), "m", nil, nil); err == nil {
		t.Error("want an error")
	}
}
