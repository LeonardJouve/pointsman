package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"uuid"

	"github.com/LeonardJouve/pointsman/store"
)

func Load() (map[uuid.UUID]Agent, error) {
	dir, err := store.Agents()
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	agents := map[uuid.UUID]Agent{}

	for _, file := range files {
		id, err := uuid.Parse(strings.TrimSuffix(filepath.Base(file), ".json"))
		if err != nil {
			return nil, err
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var a Agent
		if err := json.Unmarshal(content, &a); err != nil {
			return nil, err
		}

		agents[id] = a
	}

	return agents, nil
}

// Save writes through a temp file and a rename, so a crash never leaves a half-written agent.
func Save(id uuid.UUID, a Agent) error {
	dir, err := store.Agents()
	if err != nil {
		return err
	}

	content, err := json.Marshal(a)
	if err != nil {
		return err
	}

	file := filepath.Join(dir, id.String()+".json")
	temp := file + ".tmp"

	if err := os.WriteFile(temp, content, 0o600); err != nil {
		return err
	}

	return os.Rename(temp, file)
}
