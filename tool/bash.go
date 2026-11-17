package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
)

// pipeDelay is how long after the shell ends or is killed the tool waits for the output pipe to close.
const pipeDelay = time.Second

type bashArguments struct {
	Command string `json:"command" jsonschema:"Shell command to execute"`
	Timeout uint   `json:"timeout,omitzero" jsonschema:"Timeout in seconds (optional, no default timeout)"`
}

func Bash() (Tool, error) {
	schema, err := jsonschema.For[bashArguments](nil)
	if err != nil {
		return Tool{}, err
	}

	description := fmt.Sprintf(`Execute a bash command in the current working directory. Returns stdout and stderr.
Output is truncated to last %d lines or %dKB (whichever is hit first).
If truncated, full output is saved to a temp file. Optionally provide a timeout in seconds.`, maxLines, maxBytes/1024)

	return Tool{
		Schema: New("bash", description, schema),
		Run:    bash,
	}, nil
}

func bash(ctx context.Context, arguments string) (string, uint32) {
	var args bashArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	runCtx := ctx
	if args.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(args.Timeout)*time.Second)
		defer cancel()
	}

	var output bytes.Buffer
	cmd := exec.CommandContext(runCtx, "bash", "-c", args.Command)
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.WaitDelay = pipeDelay

	err := cmd.Run()

	code, status, err := execStatus(ctx, runCtx, err, args.Timeout)
	if err != nil {
		return fail(err)
	}

	full := strings.TrimSuffix(output.String(), "\n")
	text, note := tail(full)
	if note != "" {
		path, err := saveLog(output.Bytes())
		if err != nil {
			return fail(err)
		}

		text += fmt.Sprintf("\n\n[%s. Full output: %s]", note, path)
	}

	parts := slices.DeleteFunc([]string{text, status}, func(part string) bool { return part == "" })
	if len(parts) == 0 {
		return "(no output)", code
	}

	return strings.Join(parts, "\n\n"), code
}

// execStatus is the exit code and status line of a finished command, or err when the tool itself failed.
func execStatus(ctx, runCtx context.Context, err error, timeout uint) (uint32, string, error) {
	var code uint32
	var status string
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		code, status = 1, "Command aborted"
	case runCtx.Err() != nil:
		code, status = 1, fmt.Sprintf("Command timed out after %d seconds", timeout)
	case errors.As(err, &exit):
		code = uint32(max(exit.ExitCode(), 1))
	case err != nil && !errors.Is(err, exec.ErrWaitDelay):
		return 0, "", err
	}

	return code, status, nil
}

// saveLog writes the output to a new bash-*.log temp file and returns its path.
func saveLog(output []byte) (string, error) {
	file, err := os.CreateTemp("", "bash-*.log")
	if err != nil {
		return "", err
	}

	defer file.Close()

	if _, err := file.Write(output); err != nil {
		return "", err
	}

	return file.Name(), nil
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
