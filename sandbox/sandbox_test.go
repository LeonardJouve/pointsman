package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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

	w, err := ws.Create("python:3-slim")
	if err != nil {
		t.Fatal(err)
	}

	return w
}

func TestRun(t *testing.T) {
	w := newWorkspace(t)

	output, code, err := Run(context.Background(), w, "sh", "-c", "echo hi; echo oops >&2; echo data > out.txt")
	if err != nil || code != 0 {
		t.Fatalf("got %v %d", err, code)
	}

	if !strings.Contains(output, "hi") || !strings.Contains(output, "oops") {
		t.Errorf("got %q", output)
	}

	path, err := w.Path()
	if err != nil {
		t.Fatal(err)
	}

	if content, err := os.ReadFile(filepath.Join(path, "out.txt")); err != nil || string(content) != "data\n" {
		t.Errorf("got %q %v", content, err)
	}
}

func TestRunExitCode(t *testing.T) {
	output, code, err := Run(context.Background(), newWorkspace(t), "sh", "-c", "exit 3")
	if err != nil || code != 3 {
		t.Errorf("got %q %d %v", output, code, err)
	}
}

func TestRunBadImage(t *testing.T) {
	w := newWorkspace(t)
	w.Image = "pointsman-no-such-image.invalid/none:none"

	if _, _, err := Run(context.Background(), w, "true"); err == nil {
		t.Error("expected an error")
	}
}
