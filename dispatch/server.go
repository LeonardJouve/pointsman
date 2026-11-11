package dispatch

import (
	"context"
	"fmt"

	"github.com/LeonardJouve/pointsman/tool"
)

type server struct {
	UnimplementedDispatcherServer
	tools map[string]tool.Tool
}

func NewServer(tools []tool.Tool) *server {
	toolMap := make(map[string]tool.Tool, len(tools))
	for _, t := range tools {
		toolMap[t.Schema.Name] = t
	}

	return &server{tools: toolMap}
}

func (s *server) Call(ctx context.Context, request *Request) (*Result, error) {
	t, ok := s.tools[request.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", request.Name)
	}

	output, code := t.Run(ctx, request.Arguments)

	return &Result{Output: output, Code: code}, nil
}
