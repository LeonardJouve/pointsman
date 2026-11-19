package tool

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LeonardJouve/pointsman/workspace"
)

type Tool struct {
	Schema Schema
	Run    func(ctx context.Context, w workspace.Workspace, arguments string) (output string, code uint32)
}

// relative is the model's path for the workspace root: the model sees the workspace at workspace.Mount.
func relative(path string) (string, error) {
	if rest, ok := strings.CutPrefix(path, workspace.Mount+"/"); ok {
		return rest, nil
	}

	if strings.HasPrefix(path, "/") || filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be relative to %s: %s", workspace.Mount, path)
	}

	return path, nil
}

// fail is a tool error: the model sees the message, the dispatcher does not fail.
func fail(err error) (string, uint32) {
	return err.Error(), 1
}
