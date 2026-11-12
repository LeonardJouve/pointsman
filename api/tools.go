package api

import (
	"net/http"

	"github.com/LeonardJouve/pointsman/tool"
)

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {
	tools, err := tool.All()
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())

		return
	}

	schemas := make([]tool.Schema, len(tools))
	for i, t := range tools {
		schemas[i] = t.Schema
	}

	reply(w, http.StatusOK, schemas)
}
