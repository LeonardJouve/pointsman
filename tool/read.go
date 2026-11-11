package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

const (
	maxLines = 2000
	maxBytes = 50 * 1024
)

type readArguments struct {
	Path   string `json:"path" jsonschema:"Path to the file to read (relative or absolute)"`
	Offset int    `json:"offset,omitzero" jsonschema:"Line number to start reading from (1-indexed)"`
	Limit  int    `json:"limit,omitzero" jsonschema:"Maximum number of lines to read"`
}

func Read() (Tool, error) {
	schema, err := jsonschema.For[readArguments](nil)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Schema: New(
			"read",
			fmt.Sprintf("Read the contents of a file. Supports text files, output is truncated to %d lines or %dKB (whichever is hit first). Use offset/limit for large files. When you need the full file, continue with offset until complete.", maxLines, maxBytes/1024),
			schema,
		),
		Run: read,
	}, nil
}

func read(ctx context.Context, arguments string) (string, uint32) {
	var args readArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	content, err := os.ReadFile(resolve(args.Path))
	if err != nil {
		return fail(err)
	}

	lines := strings.Split(string(content), "\n")
	selected, start, err := window(lines, args.Offset, args.Limit)
	if err != nil {
		return fail(err)
	}

	return format(lines, selected, start), 0
}

// window is the lines from the 1-indexed offset on: at most limit of them (0 or less: no limit),
// and only whole lines under maxLines and maxBytes.
func window(lines []string, offset, limit int) ([]string, int, error) {
	start := max(offset-1, 0)
	if start >= len(lines) {
		return nil, 0, fmt.Errorf("Offset %d is beyond end of file (%d lines total)", offset, len(lines))
	}

	end := len(lines)
	if limit > 0 {
		end = min(start+limit, len(lines))
	}

	shown, size := 0, 0
	for _, line := range lines[start:end] {
		lineSize := len(line)
		if shown > 0 {
			lineSize++
		}

		if shown == maxLines || size+lineSize > maxBytes {
			break
		}

		shown++
		size += lineSize
	}

	return lines[start : start+shown], start, nil
}

// format is the selected lines, with a note on how to continue when the file has more lines after them.
func format(lines, selected []string, start int) string {
	if len(selected) == 0 {
		// TODO restore pi's notice when a bash tool exists:
		// [Line ${startLineDisplay} is ${firstLineSize}, exceeds ${formatSize(DEFAULT_MAX_BYTES)} limit. Use bash: sed -n '${startLineDisplay}p' ${path} | head -c ${DEFAULT_MAX_BYTES}]
		return fmt.Sprintf("[Line %d, exceeds %s limit.]", start+1, formatSize(maxBytes))
	}

	output := strings.Join(selected, "\n")
	last := start + len(selected)
	if last < len(lines) {
		output += fmt.Sprintf("\n\n[Showing lines %d-%d of %d. Use offset=%d to continue.]", start+1, last, len(lines), last+1)
	}

	return output
}

func formatSize(bytes int) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}
