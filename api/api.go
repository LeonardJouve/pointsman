package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
)

type Server struct {
	mutex  sync.Mutex
	agents map[uuid.UUID]agent.Agent
}

func New() *Server {
	return &Server{agents: map[uuid.UUID]agent.Agent{}}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /agents", s.createAgent)
	mux.HandleFunc("GET /agents", s.listAgents)

	return mux
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}
