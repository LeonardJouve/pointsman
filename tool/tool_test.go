package tool

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func numbered(count int) string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = "line " + strconv.Itoa(i+1)
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

func TestWrite(t *testing.T) {
	writeTool, err := Write()
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "a", "b", "file.txt")

	for _, content := range []string{"first", "second"} {
		input := arguments(t, map[string]any{"path": path, "content": content})
		output, code := writeTool.Run(context.Background(), input)
		if code != 0 || output != "Successfully wrote to "+path {
			t.Fatalf("got %q, %d", output, code)
		}

		got, err := os.ReadFile(path)
		if err != nil || string(got) != content {
			t.Errorf("got file %q, %v, want %q", got, err, content)
		}
	}
}

func TestWriteBadArguments(t *testing.T) {
	writeTool, err := Write()
	if err != nil {
		t.Fatal(err)
	}

	if _, code := writeTool.Run(context.Background(), "{"); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}

func TestSleep(t *testing.T) {
	sleepTool, err := Sleep()
	if err != nil {
		t.Fatal(err)
	}

	output, code := sleepTool.Run(context.Background(), arguments(t, map[string]any{"ms": 1}))
	if code != 0 || output != "Slept 1 ms" {
		t.Errorf("got %q, %d", output, code)
	}
}

func TestSleepBadArguments(t *testing.T) {
	sleepTool, err := Sleep()
	if err != nil {
		t.Fatal(err)
	}

	if _, code := sleepTool.Run(context.Background(), "{"); code != 1 {
		t.Errorf("got code %d, want code 1", code)
	}
}

func TestRead(t *testing.T) {
	readTool, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		return path
	}

	small := write("small", numbered(10))
	long := write("long", numbered(2500))
	wide := write("wide", strings.Repeat(strings.Repeat("x", 99)+"\n", 600))
	huge := write("huge", strings.Repeat("x", 60*1024)+"\nnext")

	tests := []struct {
		name      string
		arguments map[string]any
		output    string
		code      uint32
	}{
		{"whole file", map[string]any{"path": small}, numbered(10), 0},
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
			numbered(2000) + "\n\n[Showing lines 1-2000 of 2500. Use offset=2001 to continue.]",
			0,
		},
		{
			"byte cap",
			map[string]any{"path": wide},
			strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 99)+"\n", 512), "\n") +
				"\n\n[Showing lines 1-512 of 601. Use offset=513 to continue.]",
			0,
		},
		{"first line over byte cap", map[string]any{"path": huge}, "[Line 1, exceeds 50.0KB limit.]", 0},
		{"missing file", map[string]any{"path": filepath.Join(dir, "missing")}, "", 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := readTool.Run(context.Background(), arguments(t, test.arguments))
			if code != test.code {
				t.Fatalf("got code %d, want code %d", code, test.code)
			}

			if test.output != "" && output != test.output {
				t.Errorf("got %q, want %q", output, test.output)
			}
		})
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
		{"sleep", Sleep, []string{"ms"}, []string{"ms"}},
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
