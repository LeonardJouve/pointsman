package dispatch

import (
	"context"
	"fmt"

	"github.com/LeonardJouve/pointsman/tool"
	"github.com/LeonardJouve/pointsman/workspace"
)

type server struct {
	UnimplementedDispatcherServer
	tools      map[string]tool.Tool
	workspaces *workspace.Workspaces
}

func NewServer(tools []tool.Tool, workspaces *workspace.Workspaces) *server {
	toolMap := make(map[string]tool.Tool, len(tools))
	for _, t := range tools {
		toolMap[t.Schema.Name] = t
	}

	return &server{tools: toolMap, workspaces: workspaces}
}

func (s *server) Call(ctx context.Context, request *Request) (*Result, error) {
	t, ok := s.tools[request.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", request.Name)
	}

	w, ok := s.workspaces.Get(request.WorkspaceId)
	if !ok {
		return nil, fmt.Errorf("unknown workspace %q", request.WorkspaceId)
	}

	output, code := t.Run(ctx, w, request.Arguments)

	return &Result{Output: output, Code: code}, nil
}

func (s *server) CreateWorkspace(ctx context.Context, request *CreateWorkspaceRequest) (*CreateWorkspaceResult, error) {
	w, err := s.workspaces.Create(request.Image)
	if err != nil {
		return nil, err
	}

	return &CreateWorkspaceResult{Id: w.ID}, nil
}
