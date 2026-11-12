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

type Sessions map[uuid.UUID]*Session

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
	return s.append([]provider.Message{provider.User(text)})
}

// AppendRound writes a reply and the outputs of its tool calls, in call order, in one write: a kill leaves the
// whole round in the file or none of it, never a reply missing results. Readers see the round whole.
func (s *Session) AppendRound(reply provider.Reply, outputs []string) error {
	round := []provider.Message{reply.ToParam()}
	for i, call := range reply.ToolCalls {
		round = append(round, provider.ToolResult(call.ID, outputs[i]))
	}

	return s.append(round)
}

// append writes the messages to the file first, so the log never holds a message the file does not.
func (s *Session) append(messages []provider.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := appendMessages(s.ID, messages); err != nil {
		return err
	}

	s.messages = append(s.messages, messages...)
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
