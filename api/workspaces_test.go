package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/session"
)

// fakeDispatcher answers the workspace calls and counts them; like the dispatcher, GetWorkspace answers NotFound
// for an id not in workspaces, an empty one included.
type fakeDispatcher struct {
	dispatch.DispatcherClient
	workspaces []*dispatch.Workspace
	err        error
	calls      int
}

func (f *fakeDispatcher) CreateWorkspace(
	_ context.Context, in *dispatch.CreateWorkspaceRequest, _ ...grpc.CallOption,
) (*dispatch.Workspace, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}

	return &dispatch.Workspace{Id: "w1", Image: in.Image}, nil
}

func (f *fakeDispatcher) GetWorkspace(
	_ context.Context, in *dispatch.GetWorkspaceRequest, _ ...grpc.CallOption,
) (*dispatch.Workspace, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}

	for _, found := range f.workspaces {
		if found.Id == in.Id {
			return found, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "unknown workspace %q", in.Id)
}

func (f *fakeDispatcher) ListWorkspaces(
	context.Context, *dispatch.ListWorkspacesRequest, ...grpc.CallOption,
) (*dispatch.ListWorkspacesResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}

	return &dispatch.ListWorkspacesResult{Workspaces: f.workspaces}, nil
}

func sendWorkspace(t *testing.T, fake *fakeDispatcher, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	isolate(t, t.TempDir())

	handler := New(nil, fake, agent.Agents{}, session.Sessions{}).Handler()

	return send(handler, method, "/workspaces", body)
}

func TestCreateWorkspace(t *testing.T) {
	fake := &fakeDispatcher{}
	got := sendWorkspace(t, fake, "POST", `{"image":"alpine"}`)

	want := `{"id":"w1","image":"alpine"}`
	if got.Code != http.StatusCreated || strings.TrimSpace(got.Body.String()) != want {
		t.Errorf("got %d %s, want %s", got.Code, got.Body, want)
	}
}

func TestCreateWorkspaceBadJSON(t *testing.T) {
	fake := &fakeDispatcher{}
	got := sendWorkspace(t, fake, "POST", `{`)

	if got.Code != http.StatusBadRequest || fake.calls != 0 {
		t.Errorf("got %d, %d calls", got.Code, fake.calls)
	}
}

func TestWorkspaceDispatcherFails(t *testing.T) {
	for method, body := range map[string]string{
		"POST": `{"image":"alpine"}`,
		"GET":  "",
	} {
		got := sendWorkspace(t, &fakeDispatcher{err: errors.New("down")}, method, body)

		var failure errorResponse
		if err := json.Unmarshal(got.Body.Bytes(), &failure); err != nil || got.Code != http.StatusBadGateway ||
			failure.Error != "down" {
			t.Errorf("%s: got %d %s", method, got.Code, got.Body)
		}
	}
}

func TestListWorkspaces(t *testing.T) {
	fake := &fakeDispatcher{workspaces: []*dispatch.Workspace{{Id: "a", Image: "alpine"}, {Id: "b", Image: "debian"}}}
	got := sendWorkspace(t, fake, "GET", "")

	want := `[{"id":"a","image":"alpine"},{"id":"b","image":"debian"}]`
	if got.Code != http.StatusOK || strings.TrimSpace(got.Body.String()) != want {
		t.Errorf("got %d %s, want %s", got.Code, got.Body, want)
	}
}

func TestListNoWorkspaces(t *testing.T) {
	if got := strings.TrimSpace(sendWorkspace(t, &fakeDispatcher{}, "GET", "").Body.String()); got != "[]" {
		t.Errorf("got %s", got)
	}
}
