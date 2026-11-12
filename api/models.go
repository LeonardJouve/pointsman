package api

import "net/http"

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	ids, err := s.client.Models(r.Context())
	if err != nil {
		fail(w, http.StatusBadGateway, err.Error())

		return
	}

	reply(w, http.StatusOK, ids)
}
