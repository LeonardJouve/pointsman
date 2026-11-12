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

func New(a *agent.Agent) *Session {
	return &Session{
		ID:      uuid.New(),
		Created: time.Now(),
		Agent:   a,
		status:  Running,
		changed: make(chan struct{}),
	}
}

func (s *Session) AppendUser(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, provider.User(text))
	s.notify()
}

func (s *Session) AppendReply(reply provider.Reply) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, reply.ToParam())
	s.notify()
}

func (s *Session) AppendToolResult(callID, output string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, provider.ToolResult(callID, output))
	s.notify()
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

func (s *Session) SetStatus(status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = status
	s.notify()
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
