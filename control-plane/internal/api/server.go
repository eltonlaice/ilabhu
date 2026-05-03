package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
	"github.com/eltonlaice/ilabhu/control-plane/internal/session"
)

type Server struct {
	Catalog *catalog.Catalog
	Manager *session.Manager
	Logger  *slog.Logger
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /v1/labs", s.handleListLabs)
	mux.HandleFunc("POST /v1/sessions", s.handleCreateSession)
	mux.HandleFunc("GET /v1/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("DELETE /v1/sessions/{id}", s.handleDeleteSession)
	mux.HandleFunc("POST /v1/sessions/{id}/tasks/{task_id}/validate", s.handleValidateTask)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListLabs(w http.ResponseWriter, _ *http.Request) {
	type item struct {
		ID               string `json:"id"`
		Title            string `json:"title"`
		Exam             string `json:"exam"`
		Difficulty       string `json:"difficulty"`
		Summary          string `json:"summary"`
		EstimatedMinutes int    `json:"estimated_minutes"`
	}
	out := []item{}
	for _, m := range s.Catalog.List() {
		out = append(out, item{
			ID:               m.ID,
			Title:            m.Title,
			Exam:             m.Exam,
			Difficulty:       m.Difficulty,
			Summary:          m.Summary,
			EstimatedMinutes: m.EstimatedMinutes,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
