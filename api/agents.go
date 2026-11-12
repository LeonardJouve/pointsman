package api

import (
	"encoding/json"
	"maps"
	"net/http"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/tool"
)

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var a agent.Agent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		fail(w, http.StatusBadRequest, err.Error())

		return
	}

	for _, name := range a.Tools {
		if _, err := tool.Get(name); err != nil {
			fail(w, http.StatusBadRequest, err.Error())

			return
		}
	}

	id := uuid.New()

	if err := agent.Save(id, a); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())

		return
	}

	s.mutex.Lock()
	s.agents[id] = a
	s.mutex.Unlock()

	reply(w, http.StatusCreated, idResponse{ID: id})
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	agents := maps.Clone(s.agents)
	s.mutex.Unlock()

	reply(w, http.StatusOK, agents)
}
