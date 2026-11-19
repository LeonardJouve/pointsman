package dispatch

import (
	"context"
	"maps"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func TestCreateWorkspace(t *testing.T) {
	created, err := newServer(t).CreateWorkspace(context.Background(), &CreateWorkspaceRequest{Image: "alpine"})
	if err != nil {
		t.Fatal(err)
	}

	if created.Id == "" || created.Image != "alpine" {
		t.Errorf("got %v, want an id and image alpine", created)
	}
}

func TestGetWorkspace(t *testing.T) {
	s := newServer(t)

	created, err := s.CreateWorkspace(context.Background(), &CreateWorkspaceRequest{Image: "alpine"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetWorkspace(context.Background(), &GetWorkspaceRequest{Id: created.Id})
	if err != nil {
		t.Fatal(err)
	}

	if got.Id != created.Id || got.Image != "alpine" {
		t.Errorf("got %v, want id %s and image alpine", got, created.Id)
	}
}

func TestGetWorkspaceNotFound(t *testing.T) {
	s := newServer(t)

	for _, id := range []string{"", "missing"} {
		_, err := s.GetWorkspace(context.Background(), &GetWorkspaceRequest{Id: id})
		if status.Code(err) != codes.NotFound {
			t.Errorf("workspace id %q: got %v, want NotFound", id, err)
		}
	}
}

func TestListWorkspaces(t *testing.T) {
	s := newServer(t)

	empty, err := s.ListWorkspaces(context.Background(), &ListWorkspacesRequest{})
	if err != nil || len(empty.Workspaces) != 0 {
		t.Fatalf("got %v %v, want no workspace", empty, err)
	}

	want := map[string]string{}

	for _, image := range []string{"alpine", "python:3-slim"} {
		created, err := s.CreateWorkspace(context.Background(), &CreateWorkspaceRequest{Image: image})
		if err != nil {
			t.Fatal(err)
		}

		want[created.Id] = image
	}

	list, err := s.ListWorkspaces(context.Background(), &ListWorkspacesRequest{})
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, w := range list.Workspaces {
		got[w.Id] = w.Image
	}

	if !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCreateWorkspaceNoImage(t *testing.T) {
	if _, err := newServer(t).CreateWorkspace(context.Background(), &CreateWorkspaceRequest{}); err == nil {
		t.Error("want an error for an empty image")
	}
}
