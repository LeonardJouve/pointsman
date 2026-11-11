package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	in := "# a comment\n" +
		"\n" +
		"A = plain  \n" +
		`B="double quoted"` + "\n" +
		"C='single quoted'\n" +
		"D=has # inside\n" +
		"E=\n" +
		`F=C:\Program Files (x86)\Game` + "\n"

	got, err := parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"A": "plain",
		"B": "double quoted",
		"C": "single quoted",
		"D": "has # inside",
		"E": "",
		"F": `C:\Program Files (x86)\Game`,
	}

	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseMissingEquals(t *testing.T) {
	_, err := parse(strings.NewReader("A=1\n\nno equals sign\n"))
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("err = %v, want one naming line 3", err)
	}
}

func writeEnv(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "ENV_TEST_SET=file\nENV_TEST_UNSET=file\n")

	t.Chdir(dir)
	t.Setenv("ENV_TEST_SET", "process")
	t.Setenv("ENV_TEST_UNSET", "")
	os.Unsetenv("ENV_TEST_UNSET")

	if err := Load(".env"); err != nil {
		t.Fatal(err)
	}

	set, _ := Get("ENV_TEST_SET")
	unset, _ := Get("ENV_TEST_UNSET")

	if set != "process" || unset != "file" {
		t.Errorf("set %q, unset %q", set, unset)
	}
}

func TestLoadNoFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Load(".env"); err != nil {
		t.Errorf("Load = %v, want nil", err)
	}
}

func TestLoadParseError(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "A=1\nno equals sign\n")

	t.Chdir(dir)

	if err := Load(".env"); err == nil || !strings.HasPrefix(err.Error(), ".env: line 2") {
		t.Errorf("Load = %v, want one starting with \".env: line 2\"", err)
	}
}

func TestIsDevelopment(t *testing.T) {
	for value, want := range map[string]bool{"": true, "development": true, "production": false} {
		t.Setenv("ENVIRONMENT", value)
		if got := IsDevelopment(); got != want {
			t.Errorf("ENVIRONMENT=%q: IsDevelopment() = %v, want %v", value, got, want)
		}
	}
}
