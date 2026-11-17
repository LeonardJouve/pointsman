package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LeonardJouve/pointsman/sandbox"
	"github.com/LeonardJouve/pointsman/workspace"
	"github.com/google/jsonschema-go/jsonschema"
)

// timeoutCode is the exit code of coreutils timeout when it stops the command.
const timeoutCode = 124

type bashArguments struct {
	Command string `json:"command" jsonschema:"Shell command to execute"`
	Timeout uint   `json:"timeout,omitzero" jsonschema:"Timeout in seconds (optional, no default timeout)"`
}

func Bash() (Tool, error) {
	schema, err := jsonschema.For[bashArguments](nil)
	if err != nil {
		return Tool{}, err
	}

	description := fmt.Sprintf(`Execute a bash command in the workspace ("/work"). Returns stdout and stderr.
Output is truncated to last %d lines or %dKB (whichever is hit first).
If truncated, full output is saved to a file in the workspace. Optionally provide a timeout in seconds.
Each call runs in a new container: only files in the workspace persist between calls.`, maxLines, maxBytes/1024)

	return Tool{
		Schema: New("bash", description, schema),
		Run:    bash,
	}, nil
}

func bash(ctx context.Context, w workspace.Workspace, arguments string) (string, uint32) {
	var args bashArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	command := []string{"bash", "-c", args.Command}
	if args.Timeout > 0 {
		command = append([]string{"timeout", strconv.FormatUint(uint64(args.Timeout), 10)}, command...)
	}

	output, code, err := sandbox.Run(ctx, w, command...)

	var status string
	switch {
	case ctx.Err() != nil:
		code, status = 1, "Command aborted"
	case err != nil:
		return fail(err)
	case code == timeoutCode && args.Timeout > 0:
		code, status = 1, fmt.Sprintf("Command timed out after %d seconds", args.Timeout)
	}

	text, note := tail(strings.TrimSuffix(output, "\n"))
	if note != "" {
		name, err := saveLog(w, output)
		if err != nil {
			return fail(err)
		}

		text += fmt.Sprintf("\n\n[%s. Full output: %s]", note, name)
	}

	parts := slices.DeleteFunc([]string{text, status}, func(part string) bool { return part == "" })
	if len(parts) == 0 {
		return "(no output)", code
	}

	return strings.Join(parts, "\n\n"), code
}

// saveLog writes the output to a new bash-*.log file in the workspace and returns its name there.
func saveLog(w workspace.Workspace, output string) (string, error) {
	path, err := w.Path()
	if err != nil {
		return "", err
	}

	file, err := os.CreateTemp(path, "bash-*.log")
	if err != nil {
		return "", err
	}

	defer file.Close()

	if _, err := file.WriteString(output); err != nil {
		return "", err
	}

	return filepath.Base(file.Name()), nil
}

// tail is the end of the text: the last maxLines lines and at most maxBytes bytes, and a note on the cut
// (empty when nothing is cut).
func tail(text string) (string, string) {
	lines := strings.Split(text, "\n")

	kept, size := 0, 0
	byteLimit := false
	for i := len(lines) - 1; i >= 0 && kept < maxLines; i-- {
		lineSize := len(lines[i])
		if kept > 0 {
			lineSize++
		}

		if size+lineSize > maxBytes {
			byteLimit = true

			break
		}

		kept++
		size += lineSize
	}

	switch {
	case kept == len(lines):
		return text, ""
	case kept == 0:
		last := lines[len(lines)-1]
		part := last[len(last)-maxBytes:]
		for !utf8.ValidString(part) {
			part = part[1:]
		}

		return part, fmt.Sprintf(
			"Showing last %s of line %d (line is %s)", formatSize(len(part)), len(lines), formatSize(len(last)),
		)
	}

	note := fmt.Sprintf("Showing lines %d-%d of %d", len(lines)-kept+1, len(lines), len(lines))
	if byteLimit {
		note += fmt.Sprintf(" (%s limit)", formatSize(maxBytes))
	}

	return strings.Join(lines[len(lines)-kept:], "\n"), note
}
