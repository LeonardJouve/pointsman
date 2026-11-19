package api

import (
	"encoding/json"
	"net/http"

	"github.com/LeonardJouve/pointsman/dispatch"
)

type workspaceRequest struct {
	Image string `json:"image"`
}

type workspaceResponse struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var body workspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fail(w, http.StatusBadRequest, err.Error())

		return
	}

	created, err := s.dispatcher.CreateWorkspace(r.Context(), &dispatch.CreateWorkspaceRequest{Image: body.Image})
	if err != nil {
		fail(w, http.StatusBadGateway, err.Error())

		return
	}

	reply(w, http.StatusCreated, workspaceResponse{ID: created.Id, Image: created.Image})
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	found, err := s.dispatcher.ListWorkspaces(r.Context(), &dispatch.ListWorkspacesRequest{})
	if err != nil {
		fail(w, http.StatusBadGateway, err.Error())

		return
	}

	workspaces := make([]workspaceResponse, 0, len(found.Workspaces))
	for _, workspace := range found.Workspaces {
		workspaces = append(workspaces, workspaceResponse{ID: workspace.Id, Image: workspace.Image})
	}

	reply(w, http.StatusOK, workspaces)
}
