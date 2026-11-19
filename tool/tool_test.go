package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/LeonardJouve/pointsman/workspace"
)

func numbered(count int) string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}

	return strings.Join(lines, "\n")
}

func arguments(t *testing.T, values map[string]any) string {
	t.Helper()

	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

// folderWorkspace is a real workspace folder in a temporary config dir; no container is involved.
func folderWorkspace(t *testing.T) (workspace.Workspace, string) {
	t.Helper()

	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	workspaces, err := workspace.Load()
	if err != nil {
		t.Fatal(err)
	}

	w, err := workspaces.Create("any-image")
	if err != nil {
		t.Fatal(err)
	}

	folder, err := w.Path()
	if err != nil {
		t.Fatal(err)
	}

	return w, folder
}

func TestWrite(t *testing.T) {
	writeTool, err := Write()
	if err != nil {
		t.Fatal(err)
	}

	w, folder := folderWorkspace(t)

	for _, path := range []string{"a/b/file.txt", workspace.Mount + "/c/file.txt"} {
		for _, content := range []string{"first", "second"} {
			input := arguments(t, map[string]any{"path": path, "content": content})
			output, code := writeTool.Run(context.Background(), w, input)
			if code != 0 || output != fmt.Sprintf("Successfully wrote to %s", path) {
				t.Fatalf("got %q, %d", output, code)
			}

			got, err := os.ReadFile(filepath.Join(folder, strings.TrimPrefix(path, workspace.Mount+"/")))
			if err != nil || string(got) != content {
				t.Errorf("got file %q, %v, want %q", got, err, content)
			}
		}
	}
}

func TestWriteBadArguments(t *testing.T) {
	writeTool, err := Write()
	if err != nil {
		t.Fatal(err)
	}

	w, _ := folderWorkspace(t)

	if _, code := writeTool.Run(context.Background(), w, "{"); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}

func TestRead(t *testing.T) {
	readTool, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	w, folder := folderWorkspace(t)
	write := func(name, content string) string {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		return name
	}

	small := write("small", numbered(10))
	long := write("long", numbered(2500))
	wide := write("wide", strings.Repeat(fmt.Sprintf("%s\n", strings.Repeat("x", 99)), 600))
	huge := write("huge", fmt.Sprintf("%s\nnext", strings.Repeat("x", 60*1024)))

	tests := []struct {
		name      string
		arguments map[string]any
		output    string
		code      uint32
	}{
		{"whole file", map[string]any{"path": small}, numbered(10), 0},
		{"work prefix", map[string]any{"path": workspace.Mount + "/small"}, numbered(10), 0},
		{"offset", map[string]any{"path": small, "offset": 9}, "line 9\nline 10", 0},
		{
			"limit",
			map[string]any{"path": small, "limit": 2},
			"line 1\nline 2\n\n[Showing lines 1-2 of 10. Use offset=3 to continue.]",
			0,
		},
		{"limit zero is no limit", map[string]any{"path": small, "limit": 0}, numbered(10), 0},
		{
			"offset and limit",
			map[string]any{"path": small, "offset": 3, "limit": 2},
			"line 3\nline 4\n\n[Showing lines 3-4 of 10. Use offset=5 to continue.]",
			0,
		},
		{
			"offset past end",
			map[string]any{"path": small, "offset": 11},
			"Offset 11 is beyond end of file (10 lines total)",
			1,
		},
		{
			"line cap",
			map[string]any{"path": long},
			fmt.Sprintf("%s\n\n[Showing lines 1-2000 of 2500. Use offset=2001 to continue.]", numbered(2000)),
			0,
		},
		{
			"byte cap",
			map[string]any{"path": wide},
			fmt.Sprintf(
				"%s\n\n[Showing lines 1-512 of 601. Use offset=513 to continue.]",
				strings.TrimSuffix(strings.Repeat(fmt.Sprintf("%s\n", strings.Repeat("x", 99)), 512), "\n"),
			),
			0,
		},
		{"first line over byte cap", map[string]any{"path": huge}, "[Line 1, exceeds 50.0KB limit.]", 0},
		{"missing file", map[string]any{"path": "missing"}, "", 1},
		{"absolute path elsewhere", map[string]any{"path": "/etc/passwd"}, "path must be relative to " + workspace.Mount + ": /etc/passwd", 1},
		{"escape with ..", map[string]any{"path": "../x"}, "", 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := readTool.Run(context.Background(), w, arguments(t, test.arguments))
			if code != test.code {
				t.Fatalf("got code %d, want code %d", code, test.code)
			}

			if test.output != "" && output != test.output {
				t.Errorf("got %q, want %q", output, test.output)
			}
		})
	}
}

func TestWriteEscape(t *testing.T) {
	writeTool, err := Write()
	if err != nil {
		t.Fatal(err)
	}

	w, _ := folderWorkspace(t)

	for _, path := range []string{"/etc/x", "../x", "a/../../x"} {
		input := arguments(t, map[string]any{"path": path, "content": "x"})
		if output, code := writeTool.Run(context.Background(), w, input); code != 1 {
			t.Errorf("%s: got %q, %d, want code 1", path, output, code)
		}
	}
}

// A symlink made inside the workspace (by bash) that points outside it is not followed.
func TestSymlinkEscape(t *testing.T) {
	readTool, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	writeTool, err := Write()
	if err != nil {
		t.Fatal(err)
	}

	w, folder := folderWorkspace(t)
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(folder, "link")); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}

	if err := os.Symlink(filepath.Dir(outside), filepath.Join(folder, "dir")); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}

	for _, path := range []string{"link", "dir/secret"} {
		output, code := readTool.Run(context.Background(), w, arguments(t, map[string]any{"path": path}))
		if code != 1 {
			t.Errorf("read %s: got %q, %d, want code 1", path, output, code)
		}

		input := arguments(t, map[string]any{"path": path, "content": "changed"})
		if output, code := writeTool.Run(context.Background(), w, input); code != 1 {
			t.Errorf("write %s: got %q, %d, want code 1", path, output, code)
		}
	}

	if got, err := os.ReadFile(outside); err != nil || string(got) != "secret" {
		t.Errorf("outside file is %q, %v, want it untouched", got, err)
	}
}

func TestSchema(t *testing.T) {
	tests := []struct {
		name       string
		build      func() (Tool, error)
		properties []string
		required   []string
	}{
		{"read", Read, []string{"limit", "offset", "path"}, []string{"path"}},
		{"write", Write, []string{"content", "path"}, []string{"content", "path"}},
		{"subagent", Subagent, []string{"agent", "prompt"}, []string{"agent", "prompt"}},
		{"bash", Bash, []string{"command", "timeout"}, []string{"command"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool, err := test.build()
			if err != nil {
				t.Fatal(err)
			}

			if tool.Schema.Name != test.name {
				t.Errorf("got name %q, want %q", tool.Schema.Name, test.name)
			}

			parameters := tool.Schema.Parameters
			if got := slices.Sorted(maps.Keys(parameters.Properties)); !slices.Equal(got, test.properties) {
				t.Errorf("got properties %v, want %v", got, test.properties)
			}

			if got := slices.Sorted(slices.Values(parameters.Required)); !slices.Equal(got, test.required) {
				t.Errorf("got required %v, want %v", got, test.required)
			}
		})
	}
}
