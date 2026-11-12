package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	first, err := Dir()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { os.RemoveAll(first) })

	second, _ := Dir()
	if second != first {
		t.Errorf("got %s then %s", first, second)
	}

	if filepath.Dir(first) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(first), "pointsman-") {
		t.Errorf("got %s", first)
	}

	if info, err := os.Stat(first); err != nil || !info.IsDir() {
		t.Errorf("folder not created: %v", err)
	}
}

func TestAgents(t *testing.T) {
	folder, err := Dir()
	if err != nil {
		t.Fatal(err)
	}

	dir, err := Agents()
	if err != nil {
		t.Fatal(err)
	}

	if dir != filepath.Join(folder, "agents") {
		t.Errorf("got %s", dir)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("folder not created: %v", err)
	}
}

func TestSessions(t *testing.T) {
	folder, err := Dir()
	if err != nil {
		t.Fatal(err)
	}

	dir, err := Sessions()
	if err != nil {
		t.Fatal(err)
	}

	if dir != filepath.Join(folder, "sessions") {
		t.Errorf("got %s", dir)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("folder not created: %v", err)
	}
}
