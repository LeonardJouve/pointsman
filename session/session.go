package session

import (
	"slices"
	"sync"
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
	Agent    agent.Agent
	messages []provider.Message
	status   Status
}

func New(a agent.Agent) *Session {
	return &Session{ID: uuid.New(), Agent: a, status: Running}
}

func (s *Session) AppendUser(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, provider.User(text))
}

func (s *Session) AppendReply(reply provider.Reply) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, reply.ToParam())
}

func (s *Session) AppendToolResult(callID, output string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, provider.ToolResult(callID, output))
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
}
