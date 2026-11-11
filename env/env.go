// Package env reads settings from the `.env` file of the working directory and the process environment.
// A variable already set in the process environment wins over the file.
package env

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// Load sets the variables of the file at path that the process does not set.
// A missing file is not an error.
func Load(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}
	defer f.Close()

	vals, err := parse(f)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	for k, v := range vals {
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}

	return nil
}

// parse reads `KEY=VALUE` lines. Blank lines and lines starting with `#` are skipped, and one pair of
// surrounding quotes is stripped from a value.
func parse(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(r)

	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)

		if !ok || key == "" {
			return nil, fmt.Errorf("line %d: want NAME=value", n)
		}

		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}

		out[key] = val
	}

	return out, sc.Err()
}

// Get is the value of a setting, "" if it is not set.
func Get(name string) (string, bool) {
	value := os.Getenv(name)
	return value, value != ""
}

func IsDevelopment() bool {
	return os.Getenv("ENVIRONMENT") != "production"
}
