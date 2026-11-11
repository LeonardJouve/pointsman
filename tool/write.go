package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/jsonschema-go/jsonschema"
)

type writeArguments struct {
	Path    string `json:"path" jsonschema:"Path to the file to write (relative or absolute)"`
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

func write(ctx context.Context, arguments string) (string, uint32) {
	var args writeArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	path := resolve(args.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fail(err)
	}

	if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return fail(err)
	}

	return "Successfully wrote to " + args.Path, 0
}
