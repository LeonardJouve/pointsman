package api

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"strconv"
	"time"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/loop"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

// keepAliveInterval is how often an idle stream writes a comment line.
const keepAliveInterval = 15 * time.Second

type sessionRequest struct {
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
}

type sessionResponse struct {
	Status   session.Status     `json:"status"`
	Messages []provider.Message `json:"messages"`
}

type sessionSummary struct {
	Created time.Time      `json:"created"`
	Status  session.Status `json:"status"`
	Agent   *agent.Agent   `json:"agent"`
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var body sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fail(w, http.StatusBadRequest, err.Error())

		return
	}

	agentID, err := uuid.Parse(body.Agent)
	if err != nil {
		fail(w, http.StatusBadRequest, err.Error())

		return
	}

	s.mutex.Lock()
	a, ok := s.agents[agentID]
	s.mutex.Unlock()

	if !ok {
		fail(w, http.StatusBadRequest, "unknown agent")

		return
	}

	created := session.New(a)
	created.AppendUser(body.Prompt)

	s.mutex.Lock()
	s.sessions[created.ID] = created
	s.mutex.Unlock()

	go func() {
		if err := loop.Run(context.Background(), created, s.client, s.dispatcher); err != nil {
			log.Print(err)
		}
	}()

	reply(w, http.StatusCreated, idResponse{ID: created.ID})
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	sessions := maps.Clone(s.sessions)
	s.mutex.Unlock()

	summaries := make(map[uuid.UUID]sessionSummary, len(sessions))
	for id, found := range sessions {
		summaries[id] = sessionSummary{
			Created: found.Created,
			Status:  found.Status(),
			Agent:   found.Agent,
		}
	}

	reply(w, http.StatusOK, summaries)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		fail(w, http.StatusNotFound, "unknown session")

		return
	}

	s.mutex.Lock()
	found, ok := s.sessions[id]
	s.mutex.Unlock()

	if !ok {
		fail(w, http.StatusNotFound, "unknown session")

		return
	}

	reply(w, http.StatusOK, sessionResponse{Status: found.Status(), Messages: found.Messages()})
}

// streamSession sends the log as server-sent events, then the final status, and closes.
func (s *Server) streamSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		fail(w, http.StatusNotFound, "unknown session")

		return
	}

	s.mutex.Lock()
	found, ok := s.sessions[id]
	s.mutex.Unlock()

	if !ok {
		fail(w, http.StatusNotFound, "unknown session")

		return
	}

	next := 0
	if last, err := strconv.Atoi(r.Header.Get("Last-Event-ID")); err == nil {
		next = last + 1
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	out := http.NewResponseController(w)
	tick := time.NewTicker(keepAliveInterval)
	defer tick.Stop()

	for {
		// Order matters
		changed := found.Changed()
		status := found.Status()
		messages := found.Messages()

		next = max(0, min(next, len(messages)))

		for _, message := range messages[next:] {
			writeMessage(w, next, message)
			next++
		}

		if status != session.Running {
			writeStatus(w, status)
			out.Flush()

			return
		}

		out.Flush()

		select {
		case <-changed:
		case <-tick.C:
			writeKeepAlive(w)
			out.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
