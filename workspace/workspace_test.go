package workspace

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeonardJouve/pointsman/store"
)

func setup(t *testing.T) *Workspaces {
	t.Helper()

	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	ws, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	return ws
}

func TestCreateGet(t *testing.T) {
	ws := setup(t)

	w, err := ws.Create("alpine")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := ws.Get(w.ID)
	if !ok || got != w || w.Image != "alpine" {
		t.Errorf("got %v %v, want %v", got, ok, w)
	}

	path, err := w.Path()
	if err != nil {
		t.Fatal(err)
	}

	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Errorf("folder not created: %v", err)
	}

	if _, ok := ws.Get("missing"); ok {
		t.Error("found a missing workspace")
	}
}

func TestList(t *testing.T) {
	ws := setup(t)

	if got := ws.List(); len(got) != 0 {
		t.Errorf("got %v, want no workspace", got)
	}

	a, err := ws.Create("python:3-slim")
	if err != nil {
		t.Fatal(err)
	}

	b, err := ws.Create("alpine")
	if err != nil {
		t.Fatal(err)
	}

	got := map[Workspace]struct{}{}
	for _, w := range ws.List() {
		got[w] = struct{}{}
	}

	want := map[Workspace]struct{}{
		a: {},
		b: {},
	}
	if !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCreateNoImage(t *testing.T) {
	if _, err := setup(t).Create(""); err == nil {
		t.Error("expected an error")
	}
}

func TestLoad(t *testing.T) {
	ws := setup(t)

	a, err := ws.Create("python:3-slim")
	if err != nil {
		t.Fatal(err)
	}

	b, err := ws.Create("alpine")
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	for _, w := range []Workspace{a, b} {
		if got, ok := loaded.Get(w.ID); !ok || got != w {
			t.Errorf("got %v %v, want %v", got, ok, w)
		}
	}
}

func TestLoadBadLine(t *testing.T) {
	setup(t)

	dir, err := store.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	content := "{\"id\":\"a\",\"image\":\"b\"}\nnot json\n"
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("got %v", err)
	}
}
