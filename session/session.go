package session

import (
	"slices"
	"sync"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
)

type Session struct {
	ID    uuid.UUID
	Agent agent.Agent

	mu       sync.Mutex
	messages []provider.Message
}

func New(a agent.Agent) *Session {
	return &Session{ID: uuid.New(), Agent: a}
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
