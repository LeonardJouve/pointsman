package session

import (
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/provider"
)

type Session struct {
	ID       uuid.UUID
	Agent    agent.Agent
	Messages []provider.Message
}

func New(a agent.Agent) *Session {
	return &Session{ID: uuid.New(), Agent: a}
}

func (s *Session) AppendUser(text string) {
	s.Messages = append(s.Messages, provider.User(text))
}

func (s *Session) AppendReply(reply provider.Reply) {
	s.Messages = append(s.Messages, reply.ToParam())
}

func (s *Session) AppendToolResult(callID, output string) {
	s.Messages = append(s.Messages, provider.ToolResult(callID, output))
}
