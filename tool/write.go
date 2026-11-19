package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LeonardJouve/pointsman/workspace"
	"github.com/google/jsonschema-go/jsonschema"
)

type writeArguments struct {
	Path    string `json:"path" jsonschema:"Path to the file to write (relative to the workspace)"`
	Content string `json:"content" jsonschema:"Content to write to the file"`
}

func Write() (Tool, error) {
	schema, err := jsonschema.For[writeArguments](nil)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Schema: New(
			"write",
			"Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Automatically creates parent directories.",
			schema,
		),
		Run: write,
	}, nil
}

func write(ctx context.Context, w workspace.Workspace, arguments string) (string, uint32) {
	var args writeArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	path, err := relative(args.Path)
	if err != nil {
		return fail(err)
	}

	folder, err := w.Path()
	if err != nil {
		return fail(err)
	}

	root, err := os.OpenRoot(folder)
	if err != nil {
		return fail(err)
	}

	defer root.Close()

	if err := root.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fail(err)
	}

	if err := root.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return fail(err)
	}

	return fmt.Sprintf("Successfully wrote to %s", args.Path), 0
}
