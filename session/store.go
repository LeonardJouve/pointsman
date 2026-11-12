package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/store"
)

type header struct {
	Created time.Time `json:"created"`
	Agent   uuid.UUID `json:"agent"`
}

// kind is the key of a line's only field.
type kind string

const (
	kindHeader  kind = "header"
	kindMessage kind = "message"
	kindStatus  kind = "status"
)

// create starts the session's file with its header; it fails if the file exists.
func create(s *Session) error {
	return writeLines(s.ID, os.O_CREATE|os.O_EXCL|os.O_WRONLY, kindHeader, header{s.Created, s.Agent.ID})
}

// appendMessages writes the messages in one write, so they are all in the file or none.
func appendMessages(id uuid.UUID, messages []provider.Message) error {
	return writeLines(id, os.O_APPEND|os.O_WRONLY, kindMessage, messages...)
}

func appendStatus(id uuid.UUID, s Status) error {
	return writeLines(id, os.O_APPEND|os.O_WRONLY, kindStatus, s)
}

// writeLines writes a {"<k>": value} line for every value, all in one write.
func writeLines[T any](id uuid.UUID, flag int, k kind, values ...T) error {
	dir, err := store.Sessions()
	if err != nil {
		return err
	}

	var lines []byte
	for _, value := range values {
		line, err := json.Marshal(map[string]any{string(k): value})
		if err != nil {
			return err
		}

		lines = append(append(lines, line...), '\n')
	}

	file, err := os.OpenFile(filepath.Join(dir, id.String()+".jsonl"), flag, 0o600)
	if err != nil {
		return err
	}

	_, err = file.Write(lines)

	return errors.Join(err, file.Close())
}

// Load reads every session file; a file that cannot be loaded is an error.
// A file with no status line has not ended: its session is Running, and nothing runs it.
func Load(agents agent.Agents) (Sessions, error) {
	dir, err := store.Sessions()
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	sessions := Sessions{}

	for _, file := range files {
		s, err := load(file, agents)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}

		sessions[s.ID] = s
	}

	return sessions, nil
}

func load(file string, agents agent.Agents) (*Session, error) {
	id, err := uuid.Parse(strings.TrimSuffix(filepath.Base(file), ".jsonl"))
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")

	first, err := decode(lines[0])
	if err != nil {
		return nil, err
	}

	var h header
	if err := json.Unmarshal(first[kindHeader], &h); err != nil {
		return nil, fmt.Errorf("first line is not a header: %w", err)
	}

	a, ok := agents[h.Agent]
	if !ok {
		return nil, fmt.Errorf("unknown agent %s", h.Agent)
	}

	s := &Session{ID: id, Created: h.Created, Agent: a, status: Running, changed: make(chan struct{})}

	for _, line := range lines[1:] {
		entry, err := decode(line)
		if err != nil {
			return nil, err
		}

		for k, value := range entry {
			if err := s.restore(k, value); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

// restore applies one line of the file to a session being loaded.
func (s *Session) restore(k kind, value json.RawMessage) error {
	switch k {
	case kindMessage:
		var m provider.Message
		if err := json.Unmarshal(value, &m); err != nil {
			return err
		}

		s.messages = append(s.messages, m)

		return nil
	case kindStatus:
		return json.Unmarshal(value, &s.status)
	}

	return fmt.Errorf("unknown kind %q", k)
}

// decode splits a line into its kind and value.
func decode(line string) (map[kind]json.RawMessage, error) {
	var entry map[kind]json.RawMessage

	return entry, json.Unmarshal([]byte(line), &entry)
}
