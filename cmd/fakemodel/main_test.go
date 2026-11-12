package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go/v3"

	"github.com/LeonardJouve/pointsman/provider"
)

func send(t *testing.T, client *provider.Client, prompt string, assistants int) provider.Reply {
	t.Helper()

	messages := []provider.Message{provider.System("p"), provider.User(prompt)}
	for range assistants {
		messages = append(messages, openai.AssistantMessage("x"), provider.ToolResult("c", "ok"))
	}

	reply, err := client.Send(context.Background(), provider.Request{Model: "m", Messages: messages})
	if err != nil {
		t.Fatal(err)
	}

	return reply
}

func sleepMs(t *testing.T, reply provider.Reply) int {
	t.Helper()

	if len(reply.ToolCalls) != 1 || reply.ToolCalls[0].Function.Name != "sleep" {
		t.Fatalf("tool calls = %+v", reply.ToolCalls)
	}

	var arguments struct{ Ms int }
	if err := json.Unmarshal([]byte(reply.ToolCalls[0].Function.Arguments), &arguments); err != nil {
		t.Fatal(err)
	}

	return arguments.Ms
}

func TestSession(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /chat/completions", complete)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := provider.New(server.URL, "x")

	first := sleepMs(t, send(t, client, "17", 0))
	if first < sleepMin || first > sleepMax {
		t.Errorf("ms = %d", first)
	}

	if again := sleepMs(t, send(t, client, "17", 0)); again != first {
		t.Errorf("same prompt and turn: ms = %d, then %d", first, again)
	}

	if other := sleepMs(t, send(t, client, "18", 0)); other == first {
		t.Errorf("prompt 18 got the same ms as 17: %d", other)
	}

	if done := send(t, client, "17", rounds); done.Content != "done" || len(done.ToolCalls) != 0 {
		t.Errorf("last reply = %+v", done)
	}
}
