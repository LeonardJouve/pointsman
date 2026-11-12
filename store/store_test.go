package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgents(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir, err := Agents()
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(filepath.Dir(dir)) != "pointsman" || filepath.Base(dir) != "agents" {
		t.Errorf("got %s", dir)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("folder not created: %v", err)
	}
}
