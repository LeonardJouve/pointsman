package tool

import (
	"context"
	"path/filepath"
)

type Tool struct {
	Schema Schema
	Run    func(ctx context.Context, arguments string) (output string, code uint32)
}

// TODO move all this to workspaces

// root is where relative paths start. Confinement comes with workspaces.
const root = "."

func resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(root, path)
}

// fail is a tool error: the model sees the message, the dispatcher does not fail.
func fail(err error) (string, uint32) {
	return err.Error(), 1
}
