package tool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LeonardJouve/pointsman/env"
	"github.com/LeonardJouve/pointsman/workspace"
	"github.com/google/jsonschema-go/jsonschema"
)

type subagentArguments struct {
	Agent  string `json:"agent" jsonschema:"Id of the agent to run"`
	Prompt string `json:"prompt" jsonschema:"What the agent must do"`
}

type sessionReply struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type sessionMessage struct {
	Content string `json:"content"`
}

type sessionStatus struct {
	Status string `json:"status"`
}

// maxFrameBytes is the longest SSE line the scanner accepts: a message frame holds a whole message.
const maxFrameBytes = 1 << 20

func Subagent() (Tool, error) {
	schema, err := jsonschema.For[subagentArguments](nil)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Schema: New("subagent", "Run another agent on a prompt and return its final answer.", schema),
		Run:    subagent,
	}, nil
}

// subagent starts a session on the router, waits for it to end and returns its last assistant text.
func subagent(ctx context.Context, w workspace.Workspace, arguments string) (string, uint32) {
	var args subagentArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	addr, ok := env.Get("ROUTER_ADDR")
	if !ok {
		return fail(errors.New("ROUTER_ADDR is not set"))
	}

	router := fmt.Sprintf("http://%s", addr)

	id, err := createSession(ctx, router, args)
	if err != nil {
		return fail(err)
	}

	answer, err := waitSession(ctx, router, id)
	if err != nil {
		return fail(err)
	}

	return answer, 0
}

func createSession(ctx context.Context, router string, args subagentArguments) (string, error) {
	body, err := json.Marshal(args)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/sessions", router)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var reply sessionReply
	_ = json.NewDecoder(response.Body).Decode(&reply)

	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create session: %s: %s", response.Status, reply.Error)
	}

	return reply.ID, nil
}

// waitSession reads the events of the session until its status frame and returns the last message text.
func waitSession(ctx context.Context, router string, id string) (string, error) {
	url := fmt.Sprintf("%s/sessions/%s/events", router, id)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("events: %s", response.Status)
	}

	var event, answer string

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(nil, maxFrameBytes)

	for scanner.Scan() {
		line := scanner.Text()

		if name, ok := strings.CutPrefix(line, "event: "); ok {
			event = name
			continue
		}

		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}

		switch event {
		case "message":
			var message sessionMessage
			if err := json.Unmarshal([]byte(data), &message); err != nil {
				return "", err
			}

			answer = message.Content
		case "status":
			var status sessionStatus
			if err := json.Unmarshal([]byte(data), &status); err != nil {
				return "", err
			}

			if status.Status == "failed" {
				return "", fmt.Errorf("subagent session %s failed", id)
			}

			return answer, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("events ended before the status")
}
