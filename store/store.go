package store

import (
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, "pointsman")

	return dir, os.MkdirAll(dir, 0o700)
}

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
