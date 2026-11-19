package workspace

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"uuid"

	"github.com/LeonardJouve/pointsman/store"
)

const file = "workspaces.jsonl"

// Mount is where the workspace folder appears in the containers.
const Mount = "/work"

// Workspace is a host folder mounted at Mount in each container of its tools.
type Workspace struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

// Path is the workspace's folder.
func (w Workspace) Path() (string, error) {
	dir, err := store.Workspaces()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, w.ID), nil
}

type Workspaces struct {
	mutex      sync.Mutex
	workspaces map[string]Workspace
}

// Load reads the workspaces file; a missing file is no workspace.
func Load() (*Workspaces, error) {
	dir, err := store.Workspaces()
	if err != nil {
		return nil, err
	}

	ws := &Workspaces{
		workspaces: map[string]Workspace{},
	}

	f, err := os.Open(filepath.Join(dir, file))
	if os.IsNotExist(err) {
		return ws, nil
	}

	if err != nil {
		return nil, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	for line := 1; scanner.Scan(); line++ {
		var w Workspace
		if err := json.Unmarshal(scanner.Bytes(), &w); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", file, line, err)
		}

		ws.workspaces[w.ID] = w
	}

	return ws, scanner.Err()
}

// Create makes a workspace and its folder; the image is required.
func (ws *Workspaces) Create(image string) (Workspace, error) {
	if image == "" {
		return Workspace{}, errors.New("image is required")
	}

	w := Workspace{
		ID:    uuid.New().String(),
		Image: image,
	}

	path, err := w.Path()
	if err != nil {
		return Workspace{}, err
	}

	line, err := json.Marshal(w)
	if err != nil {
		return Workspace{}, err
	}

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if err := os.Mkdir(path, 0o700); err != nil {
		return Workspace{}, err
	}

	f, err := os.OpenFile(filepath.Join(filepath.Dir(path), file), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return Workspace{}, err
	}

	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()

		return Workspace{}, err
	}

	if err := f.Close(); err != nil {
		return Workspace{}, err
	}

	ws.workspaces[w.ID] = w

	return w, nil
}

// List returns every workspace, in no particular order.
func (ws *Workspaces) List() []Workspace {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	list := make([]Workspace, 0, len(ws.workspaces))
	for _, w := range ws.workspaces {
		list = append(list, w)
	}

	return list
}

func (ws *Workspaces) Get(id string) (Workspace, bool) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	w, ok := ws.workspaces[id]

	return w, ok
}
