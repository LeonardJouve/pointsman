package agent

import (
	"reflect"
	"testing"
	"uuid"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestSaveLoad(t *testing.T) {
	isolate(t)

	id := uuid.New()
	a := New("n", "m", "p", []string{"write"}, Config{Effort: "low", MaxTokens: 10})

	if err := Save(id, a); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if want := map[uuid.UUID]Agent{id: a}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadEmpty(t *testing.T) {
	isolate(t)

	got, err := Load()
	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v", got, err)
	}
}
