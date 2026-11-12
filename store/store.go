package store

import (
	"os"
	"path/filepath"
	"sync"
)

// Dir is a new temporary folder per process.
var Dir = sync.OnceValues(func() (string, error) {
	return os.MkdirTemp("", "pointsman-")
})

func Agents() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	agents := filepath.Join(dir, "agents")

	return agents, os.MkdirAll(agents, 0o700)
}

func Sessions() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	sessions := filepath.Join(dir, "sessions")

	return sessions, os.MkdirAll(sessions, 0o700)
}
