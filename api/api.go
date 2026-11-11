package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"uuid"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

type Server struct {
	mutex      sync.Mutex
	client     *provider.Client
	dispatcher dispatch.DispatcherClient
	agents     map[uuid.UUID]agent.Agent
	sessions   map[uuid.UUID]*session.Session
}

type idResponse struct {
	ID uuid.UUID `json:"id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(client *provider.Client, dispatcher dispatch.DispatcherClient) *Server {
	return &Server{
		client:     client,
		dispatcher: dispatcher,
		agents:     map[uuid.UUID]agent.Agent{},
		sessions:   map[uuid.UUID]*session.Session{},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /agents", s.createAgent)
	mux.HandleFunc("GET /agents", s.listAgents)
	mux.HandleFunc("POST /sessions", s.createSession)
	mux.HandleFunc("GET /sessions/{id}", s.getSession)

	return mux
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, errorResponse{Error: message})
}
