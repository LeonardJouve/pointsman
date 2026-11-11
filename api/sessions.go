package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"uuid"

	"github.com/LeonardJouve/pointsman/loop"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

type sessionRequest struct {
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
}

type sessionResponse struct {
	Status   session.Status     `json:"status"`
	Messages []provider.Message `json:"messages"`
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
