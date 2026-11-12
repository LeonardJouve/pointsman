package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestListTools(t *testing.T) {
	got := do(t, "GET", "/tools", "")
	if got.Code != http.StatusOK {
		t.Fatalf("got %d %s", got.Code, got.Body)
	}

	for _, name := range []string{`"Name":"read"`, `"Name":"write"`} {
		if !strings.Contains(got.Body.String(), name) {
			t.Errorf("missing %s in %s", name, got.Body)
		}
	}
}
