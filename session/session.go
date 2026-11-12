package session

import (
	"slices"
	"sync"
	"time"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
)

type Status string

const (
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
)

type Session struct {
	mu       sync.Mutex
	ID       uuid.UUID
	Created  time.Time
	Agent    *agent.Agent
	messages []provider.Message
	status   Status
	changed  chan struct{}
}

func New(a *agent.Agent) (*Session, error) {
	s := &Session{
		ID:      uuid.New(),
		Created: time.Now(),
		Agent:   a,
		status:  Running,
		changed: make(chan struct{}),
	}

	return s, create(s)
}

func (s *Session) AppendUser(text string) error {
	return s.append(provider.User(text))
}

func (s *Session) AppendReply(reply provider.Reply) error {
	return s.append(reply.ToParam())
}

func (s *Session) AppendToolResult(callID, output string) error {
	return s.append(provider.ToolResult(callID, output))
}

// append writes the message to the file first, so the log never holds a message the file does not.
func (s *Session) append(message provider.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := appendMessage(s.ID, message); err != nil {
		return err
	}

	s.messages = append(s.messages, message)
	s.notify()

	return nil
}

// Messages returns a copy of the log, safe to read while the loop appends.
func (s *Session) Messages() []provider.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.messages)
}

func (s *Session) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.status
}

// SetStatus takes a final status and writes it to the file; a file with no status line has not ended.
// The memory is set and the readers woken even if the write fails, so a reader never waits on Running.
func (s *Session) SetStatus(status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = status
	s.notify()

	return appendStatus(s.ID, status)
}

// Changed returns a channel that is closed at the next change.
func (s *Session) Changed() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.changed
}

// notify wakes the readers of Changed; the caller holds the lock.
func (s *Session) notify() {
	close(s.changed)
	s.changed = make(chan struct{})
}
