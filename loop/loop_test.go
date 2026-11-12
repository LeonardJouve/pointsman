package loop

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

const (
	finalReply = `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`
	callsReply = `{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[` +
		`{"id":"a","type":"function","function":{"name":"look","arguments":"1"}},` +
		`{"id":"b","type":"function","function":{"name":"look","arguments":"2"}}]}}]}`
	truncatedReply = `{"choices":[{"finish_reason":"length","message":{"role":"assistant","content":"x"}}]}`
)

type fakeDispatcher func(request *dispatch.Request) (*dispatch.Result, error)

func (f fakeDispatcher) Call(
	_ context.Context,
	request *dispatch.Request,
	_ ...grpc.CallOption,
) (*dispatch.Result, error) {
	return f(request)
}

func echo(request *dispatch.Request) (*dispatch.Result, error) {
	return &dispatch.Result{Output: "out" + request.Arguments}, nil
}

// serve returns a client whose model gives the replies in order.
func serve(t *testing.T, replies ...string) *provider.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(replies[0]))
		replies = replies[1:]
	}))
	t.Cleanup(server.Close)

	return provider.New(server.URL, "key")
}

func start(t *testing.T) *session.Session {
	t.Helper()

	s, err := session.New(&agent.Agent{})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.AppendUser("go"); err != nil {
		t.Fatal(err)
	}

	return s
}

// roles returns the role of each message in the log.
func roles(s *session.Session) string {
	var out []string
	for _, m := range s.Messages() {
		switch {
		case m.OfUser != nil:
			out = append(out, "user")
		case m.OfAssistant != nil:
			out = append(out, "assistant")
		case m.OfTool != nil:
			out = append(out, "tool")
		}
	}

	return strings.Join(out, " ")
}

func toolContent(s *session.Session, i int) string {
	return s.Messages()[i].OfTool.Content.OfString.Value
}

func TestRunNoToolCalls(t *testing.T) {
	s := start(t)

	if err := Run(context.Background(), s, serve(t, finalReply), fakeDispatcher(echo)); err != nil {
		t.Fatal(err)
	}

	if got := roles(s); got != "user assistant" {
		t.Errorf("roles = %s", got)
	}

	if got := s.Status(); got != session.Succeeded {
		t.Errorf("status = %s", got)
	}
}

func TestRunToolCalls(t *testing.T) {
	s := start(t)
	dispatcher := fakeDispatcher(func(request *dispatch.Request) (*dispatch.Result, error) {
		if request.SessionId != s.ID.String() || request.Name != "look" {
			t.Errorf("request = %+v", request)
		}

		return echo(request)
	})

	if err := Run(context.Background(), s, serve(t, callsReply, finalReply), dispatcher); err != nil {
		t.Fatal(err)
	}

	if got := roles(s); got != "user assistant tool tool assistant" {
		t.Fatalf("roles = %s", got)
	}

	if s.Messages()[2].OfTool.ToolCallID != "a" || toolContent(s, 2) != "out1" ||
		s.Messages()[3].OfTool.ToolCallID != "b" || toolContent(s, 3) != "out2" {
		t.Errorf("results = %+v, %+v", s.Messages()[2], s.Messages()[3])
	}

	if got := s.Status(); got != session.Succeeded {
		t.Errorf("status = %s", got)
	}
}

func TestRunNonZeroCode(t *testing.T) {
	s := start(t)
	dispatcher := fakeDispatcher(func(request *dispatch.Request) (*dispatch.Result, error) {
		if request.CallId == "a" {
			return &dispatch.Result{Output: "boom", Code: 2}, nil
		}

		return &dispatch.Result{Code: 3}, nil
	})

	if err := Run(context.Background(), s, serve(t, callsReply, finalReply), dispatcher); err != nil {
		t.Fatal(err)
	}

	if got := toolContent(s, 2); got != "boom\nCommand exited with code 2" {
		t.Errorf("output = %q", got)
	}

	if got := toolContent(s, 3); got != "\nCommand exited with code 3" {
		t.Errorf("output = %q", got)
	}
}

func TestRunDispatcherError(t *testing.T) {
	s := start(t)
	down := errors.New("down")
	dispatcher := fakeDispatcher(func(request *dispatch.Request) (*dispatch.Result, error) {
		if request.CallId == "b" {
			return nil, down
		}

		return echo(request)
	})

	if err := Run(context.Background(), s, serve(t, callsReply), dispatcher); !errors.Is(err, down) {
		t.Errorf("err = %v", err)
	}

	if got := roles(s); got != "user" {
		t.Errorf("roles = %s", got)
	}

	if got := s.Status(); got != session.Failed {
		t.Errorf("status = %s", got)
	}
}
