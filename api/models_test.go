package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestListModels(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"ids":   {`{"object":"list","data":[{"id":"a","object":"model"},{"id":"b","object":"model"}]}`, `["a","b"]`},
		"empty": {`{"object":"list","data":[]}`, `[]`},
	} {
		got := send(serve(t, http.StatusOK, tc.body), "GET", "/models", "")
		if got.Code != http.StatusOK || strings.TrimSpace(got.Body.String()) != tc.want {
			t.Errorf("%s: got %d %s, want %s", name, got.Code, got.Body, tc.want)
		}
	}
}

func TestListModelsProviderFailure(t *testing.T) {
	got := send(serve(t, http.StatusInternalServerError, `{"error":{"message":"down"}}`), "GET", "/models", "")
	if got.Code != http.StatusBadGateway || !strings.Contains(got.Body.String(), `"error"`) {
		t.Errorf("got %d %s", got.Code, got.Body)
	}
}
