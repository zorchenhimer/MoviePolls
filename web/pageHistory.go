package web

import (
	"net/http"

	"github.com/zorchenhimer/MoviePolls/models"
)

// List of past cycles
func (s *webServer) handlerPageHistory(w http.ResponseWriter, r *http.Request) {
	past, err := s.backend.GetPastCycles(0, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)
		s.l.Error("Unable to get past cycles: ", err)
		return
	}

	data := struct {
		dataPageBase
		Cycles []*models.Cycle
	}{
		dataPageBase: s.newPageBase("Cycle History", w, r),
		Cycles:       past,
	}

	if err := s.executeTemplate(w, "history", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
