package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
	"github.com/eltonlaice/ilabhu/control-plane/internal/session"
)

// SessionManager is the slice of session.Manager that the HTTP layer needs.
// Defining it as an interface lets the api package be tested with a fake
// implementation while keeping session.Manager free of HTTP concerns.
type SessionManager interface {
	Start(ctx context.Context, examID string, input session.StartInput) (*session.Session, error)
	Get(id string) (*session.Session, error)
	Destroy(ctx context.Context, id string, input session.StartInput) error
}

type Server struct {
	Catalog *catalog.Catalog
	Manager SessionManager
	Logger  *slog.Logger
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /v1/exams", s.handleListExams)
	mux.HandleFunc("GET /v1/exams/{id...}", s.handleGetExam)
	mux.HandleFunc("POST /v1/sessions", s.handleCreateSession)
	mux.HandleFunc("GET /v1/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("DELETE /v1/sessions/{id}", s.handleDeleteSession)
	mux.HandleFunc("POST /v1/sessions/{id}/tasks/{task_id}/validate", s.handleValidateTask)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListExams(w http.ResponseWriter, _ *http.Request) {
	type item struct {
		ID               string   `json:"id"`
		Title            string   `json:"title"`
		Exam             string   `json:"exam"`
		Difficulty       string   `json:"difficulty"`
		Summary          string   `json:"summary"`
		EstimatedMinutes int      `json:"estimated_minutes"`
		TimeLimitMinutes int      `json:"time_limit_minutes"`
		PassingScore     int      `json:"passing_score"`
		Providers        []string `json:"providers"`
	}
	out := []item{}
	for _, m := range s.Catalog.List() {
		providers := make([]string, 0, len(m.Infrastructure.Providers))
		for k := range m.Infrastructure.Providers {
			providers = append(providers, k)
		}
		out = append(out, item{
			ID:               m.ID,
			Title:            m.Title,
			Exam:             m.Exam,
			Difficulty:       m.Difficulty,
			Summary:          m.Summary,
			EstimatedMinutes: m.EstimatedMinutes,
			TimeLimitMinutes: m.TimeLimitMinutes,
			PassingScore:     m.PassingScore,
			Providers:        providers,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetExam(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	exam, ok := s.Catalog.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "exam not found")
		return
	}
	type taskOut struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Domain       string `json:"domain,omitempty"`
		Weight       int    `json:"weight,omitempty"`
		Instructions string `json:"instructions"`
	}
	tasks := make([]taskOut, 0, len(exam.Tasks))
	for _, t := range exam.Tasks {
		tasks = append(tasks, taskOut{
			ID:           t.ID,
			Title:        t.Title,
			Domain:       t.Domain,
			Weight:       t.Weight,
			Instructions: t.Instructions,
		})
	}
	providers := make([]string, 0, len(exam.Infrastructure.Providers))
	for k := range exam.Infrastructure.Providers {
		providers = append(providers, k)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                 exam.ID,
		"version":            exam.Version,
		"exam":               exam.Exam,
		"title":              exam.Title,
		"summary":            exam.Summary,
		"difficulty":         exam.Difficulty,
		"estimated_minutes":  exam.EstimatedMinutes,
		"time_limit_minutes": exam.TimeLimitMinutes,
		"passing_score":      exam.PassingScore,
		"domains":            exam.Domains,
		"instructions":       exam.Instructions,
		"tasks":              tasks,
		"providers":          providers,
		"infrastructure": map[string]any{
			"ttl_minutes": exam.Infrastructure.TTLMinutes,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
