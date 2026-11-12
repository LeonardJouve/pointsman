package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
	"uuid"

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
	return writeLine(s.ID, os.O_CREATE|os.O_EXCL|os.O_WRONLY, kindHeader, header{s.Created, s.Agent.ID})
}

func appendMessage(id uuid.UUID, m provider.Message) error {
	return writeLine(id, os.O_APPEND|os.O_WRONLY, kindMessage, m)
}

func appendStatus(id uuid.UUID, s Status) error {
	return writeLine(id, os.O_APPEND|os.O_WRONLY, kindStatus, s)
}

// writeLine writes {"<k>": value} and a newline in one write.
func writeLine(id uuid.UUID, flag int, k kind, value any) error {
	dir, err := store.Sessions()
	if err != nil {
		return err
	}

	line, err := json.Marshal(map[string]any{string(k): value})
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(dir, id.String()+".jsonl"), flag, 0o600)
	if err != nil {
		return err
	}

	_, err = file.Write(append(line, '\n'))

	return errors.Join(err, file.Close())
}
