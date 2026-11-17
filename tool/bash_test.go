package tool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LeonardJouve/pointsman/workspace"
)

func newWorkspace(t *testing.T) workspace.Workspace {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not on PATH")
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon not answering")
	}

	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	ws, err := workspace.Load()
	if err != nil {
		t.Fatal(err)
	}

	w, err := ws.Create("debian:stable-slim")
	if err != nil {
		t.Fatal(err)
	}

	return w
}

func TestBash(t *testing.T) {
	w := newWorkspace(t)

	bashTool, err := Bash()
	if err != nil {
		t.Fatal(err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		command string
		timeout int
		output  string
		code    uint32
	}{
		{"both streams", context.Background(), "echo out; echo err >&2", 0, "out\nerr", 0},
		{"exit code", context.Background(), "echo bad; exit 3", 0, "bad", 3},
		{"no output", context.Background(), "true", 0, "(no output)", 0},
		{"timeout", context.Background(), "echo start; sleep 5", 1, "start\n\nCommand timed out after 1 seconds", 1},
		{"timeout without output", context.Background(), "sleep 5", 1, "Command timed out after 1 seconds", 1},
		{"cancelled", cancelled, "sleep 5", 0, "Command aborted", 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := arguments(t, map[string]any{"command": test.command, "timeout": test.timeout})
			output, code := bashTool.Run(test.ctx, w, input)
			if output != test.output || code != test.code {
				t.Errorf("got %q, %d, want %q, %d", output, code, test.output, test.code)
			}
		})
	}
}

func TestBashPersistence(t *testing.T) {
	w := newWorkspace(t)

	bashTool, err := Bash()
	if err != nil {
		t.Fatal(err)
	}

	write := arguments(t, map[string]any{"command": "echo kept > a.txt"})
	if output, code := bashTool.Run(context.Background(), w, write); code != 0 {
		t.Fatalf("got code %d, output %q", code, output)
	}

	output, code := bashTool.Run(context.Background(), w, arguments(t, map[string]any{"command": "cat a.txt"}))
	if output != "kept" || code != 0 {
		t.Errorf("got %q, %d, want %q, 0", output, code, "kept")
	}
}

func TestBashTruncation(t *testing.T) {
	w := newWorkspace(t)

	path, err := w.Path()
	if err != nil {
		t.Fatal(err)
	}

	bashTool, err := Bash()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		command string
		tail    string
		note    string
		full    int
	}{
		{
			"lines",
			"seq 1 2500 | sed 's/^/line /'",
			"line 2500",
			`\[Showing lines 501-2500 of 2500\. Full output: (bash-.+\.log)\]`,
			2500,
		},
		{
			"bytes",
			`yes $(head -c 99 /dev/zero | tr '\0' x) | head -n 600`,
			strings.Repeat("x", 99),
			`\[Showing lines 89-600 of 600 \(50\.0KB limit\)\. Full output: (bash-.+\.log)\]`,
			600,
		},
		{
			"one long line",
			`head -c 60000 /dev/zero | tr '\0' x; echo`,
			strings.Repeat("x", 100),
			`\[Showing last 50\.0KB of line 1 \(line is 58\.6KB\)\. Full output: (bash-.+\.log)\]`,
			1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := bashTool.Run(context.Background(), w, arguments(t, map[string]any{"command": test.command}))
			if code != 0 {
				t.Fatalf("got code %d, output %q", code, output)
			}

			match := regexp.MustCompile(test.note).FindStringSubmatch(output)
			if match == nil {
				t.Fatalf("no note in %q", output[max(len(output)-200, 0):])
			}

			text, _, _ := strings.Cut(output, "\n\n[Showing")
			if !strings.HasSuffix(text, test.tail) {
				t.Errorf("got tail %q, want %q", text[max(len(text)-20, 0):], test.tail)
			}

			full, err := os.ReadFile(filepath.Join(path, match[1]))
			if err != nil {
				t.Fatal(err)
			}

			if got := strings.Count(string(full), "\n"); got != test.full {
				t.Errorf("got %d lines in the log file, want %d", got, test.full)
			}
		})
	}
}

func TestBashBadArguments(t *testing.T) {
	bashTool, err := Bash()
	if err != nil {
		t.Fatal(err)
	}

	if _, code := bashTool.Run(context.Background(), workspace.Workspace{}, "{"); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}
