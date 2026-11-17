package dispatch

import (
	"context"
	"testing"

	"github.com/LeonardJouve/pointsman/tool"
	"github.com/LeonardJouve/pointsman/workspace"
)

func tools() []tool.Tool {
	return []tool.Tool{
		{
			Schema: tool.Schema{Name: "echo"},
			Run: func(ctx context.Context, w workspace.Workspace, arguments string) (string, uint32) {
				return w.ID + " " + w.Image + " " + arguments, 3
			},
		},
	}
}

func newServer(t *testing.T) *server {
	t.Helper()

	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	workspaces, err := workspace.Load()
	if err != nil {
		t.Fatal(err)
	}

	return NewServer(tools(), workspaces)
}

func TestCall(t *testing.T) {
	s := newServer(t)

	created, err := s.CreateWorkspace(context.Background(), &CreateWorkspaceRequest{Image: "alpine"})
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.Call(context.Background(), &Request{Name: "echo", Arguments: `{"a":1}`, WorkspaceId: created.Id})
	if err != nil {
		t.Fatal(err)
	}

	want := created.Id + ` alpine {"a":1}`
	if result.Output != want || result.Code != 3 {
		t.Errorf("got output %q code %d, want %q code 3", result.Output, result.Code, want)
	}
}

func TestCallUnknownTool(t *testing.T) {
	if _, err := newServer(t).Call(context.Background(), &Request{Name: "missing"}); err == nil {
		t.Error("want an error for an unknown tool")
	}
}

func TestCallBadWorkspace(t *testing.T) {
	s := newServer(t)

	for _, id := range []string{"", "missing"} {
		if _, err := s.Call(context.Background(), &Request{Name: "echo", WorkspaceId: id}); err == nil {
			t.Errorf("want an error for workspace id %q", id)
		}
	}
}

func TestCreateWorkspaceNoImage(t *testing.T) {
	if _, err := newServer(t).CreateWorkspace(context.Background(), &CreateWorkspaceRequest{}); err == nil {
		t.Error("want an error for an empty image")
	}
}
