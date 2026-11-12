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
