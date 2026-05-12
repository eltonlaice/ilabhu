package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
	"github.com/eltonlaice/ilabhu/control-plane/internal/session"
)

// fakeManager is a SessionManager double for handler tests.
type fakeManager struct {
	startFn   func(ctx context.Context, examID string, in session.StartInput) (*session.Session, error)
	getFn     func(id string) (*session.Session, error)
	destroyFn func(ctx context.Context, id string, in session.StartInput) error
}

func (f *fakeManager) Start(ctx context.Context, examID string, in session.StartInput) (*session.Session, error) {
	return f.startFn(ctx, examID, in)
}
func (f *fakeManager) Get(id string) (*session.Session, error) { return f.getFn(id) }
func (f *fakeManager) Destroy(ctx context.Context, id string, in session.StartInput) error {
	return f.destroyFn(ctx, id, in)
}

func newTestServer(t *testing.T, cat *catalog.Catalog, mgr SessionManager) *Server {
	t.Helper()
	return &Server{
		Catalog: cat,
		Manager: mgr,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func emptyCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const fixtureManifest = `schema_version: 1
id: cka/example
version: 3
exam: CKA
title: Example
summary: x
difficulty: easy
estimated_minutes: 7
time_limit_minutes: 60
passing_score: 66
domains:
  - name: Workloads
    weight: 100
infrastructure:
  ttl_minutes: 90
  providers:
    aws:
      module: ./aws
access:
  kind: kubeconfig
  output: kubeconfig
instructions: do it.
tasks:
  - id: t1
    title: First task
    domain: Workloads
    weight: 100
    instructions: First instructions.
    validations: []
`

func writeFixture(t *testing.T) *catalog.Catalog {
	t.Helper()
	root := t.TempDir()
	labDir := filepath.Join(root, "cka", "example")
	if err := os.MkdirAll(labDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(labDir, "exam.yaml"), []byte(fixtureManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestListExams_ReflectsCatalog(t *testing.T) {
	cat := writeFixture(t)
	srv := newTestServer(t, cat, &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/exams", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var exams []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &exams); err != nil {
		t.Fatal(err)
	}
	if len(exams) != 1 {
		t.Fatalf("got %d exams, want 1", len(exams))
	}
	if exams[0]["id"] != "cka/example" {
		t.Errorf("id = %v", exams[0]["id"])
	}
	if exams[0]["passing_score"].(float64) != 66 {
		t.Errorf("passing_score = %v", exams[0]["passing_score"])
	}
	providers, ok := exams[0]["providers"].([]any)
	if !ok || len(providers) != 1 || providers[0] != "aws" {
		t.Errorf("providers = %v", exams[0]["providers"])
	}
}

func TestGetExam_Success(t *testing.T) {
	cat := writeFixture(t)
	srv := newTestServer(t, cat, &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/exams/cka/example", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["id"] != "cka/example" {
		t.Errorf("id = %v", resp["id"])
	}
	tasks, ok := resp["tasks"].([]any)
	if !ok || len(tasks) != 1 {
		t.Fatalf("tasks not present or wrong length: %v", resp["tasks"])
	}
	first := tasks[0].(map[string]any)
	if first["id"] != "t1" || first["title"] != "First task" || first["domain"] != "Workloads" {
		t.Errorf("task fields wrong: %v", first)
	}
	infra := resp["infrastructure"].(map[string]any)
	if infra["ttl_minutes"].(float64) != 90 {
		t.Errorf("ttl_minutes = %v", infra["ttl_minutes"])
	}
	providers, ok := resp["providers"].([]any)
	if !ok || len(providers) != 1 {
		t.Fatalf("providers wrong: %v", resp["providers"])
	}
}

func TestGetExam_NotFound(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/exams/cka/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestCreateSession_Success(t *testing.T) {
	called := false
	mgr := &fakeManager{
		startFn: func(_ context.Context, examID string, in session.StartInput) (*session.Session, error) {
			called = true
			if examID != "cka/example" || in.Provider != "aws" || in.AWS == nil || in.AWS.RoleARN != "arn:aws:iam::1:role/r" {
				t.Errorf("unexpected args: %q %+v", examID, in)
			}
			return &session.Session{
				ID:        "sess-123",
				ExamID:    examID,
				Provider:  in.Provider,
				Status:    session.StatusProvisioning,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"exam_id":"cka/example","provider":"aws","aws":{"role_arn":"arn:aws:iam::1:role/r","external_id":"xid"}}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions", body))

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Error("Start not called")
	}
	var sess session.Session
	if err := json.Unmarshal(rr.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.ID != "sess-123" || sess.Provider != "aws" {
		t.Errorf("sess = %+v", sess)
	}
}

func TestCreateSession_SuccessDigitalOcean(t *testing.T) {
	mgr := &fakeManager{
		startFn: func(_ context.Context, examID string, in session.StartInput) (*session.Session, error) {
			if examID != "cka/example" || in.Provider != "digitalocean" || in.DigitalOcean == nil || in.DigitalOcean.Token != "dop_v1_xyz" {
				t.Errorf("unexpected args: %q %+v", examID, in)
			}
			return &session.Session{
				ID:        "sess-do-1",
				ExamID:    examID,
				Provider:  in.Provider,
				Status:    session.StatusProvisioning,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"exam_id":"cka/example","provider":"digitalocean","digitalocean":{"token":"dop_v1_xyz"}}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions", body))

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateSession_RejectsMissingExamID(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	body := strings.NewReader(`{"provider":"aws","aws":{"role_arn":"r"}}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions", body))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateSession_RejectsMissingProvider(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	body := strings.NewReader(`{"exam_id":"cka/example"}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions", body))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateSession_RejectsInvalidJSON(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader("not json")))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateSession_ManagerErrorIsBadRequest(t *testing.T) {
	mgr := &fakeManager{
		startFn: func(context.Context, string, session.StartInput) (*session.Session, error) {
			return nil, errors.New("aws.role_arn is required")
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"exam_id":"cka/example","provider":"aws"}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions", body))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestGetSession_Success(t *testing.T) {
	mgr := &fakeManager{
		getFn: func(id string) (*session.Session, error) {
			if id != "abc" {
				t.Errorf("id = %q", id)
			}
			return &session.Session{
				ID:         "abc",
				ExamID:     "cka/example",
				Provider:   "aws",
				Status:     session.StatusReady,
				Kubeconfig: []byte("apiVersion: v1\nkind: Config"),
				Outputs:    map[string]any{"public_ip": "1.2.3.4"},
			}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/sessions/abc", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["kubeconfig_b64"] == nil || resp["kubeconfig_b64"] == "" {
		t.Error("kubeconfig_b64 missing")
	}
	if resp["status"] != string(session.StatusReady) {
		t.Errorf("status field = %v, want %v", resp["status"], session.StatusReady)
	}
	if resp["exam_id"] != "cka/example" || resp["provider"] != "aws" {
		t.Errorf("exam_id/provider wrong: %v %v", resp["exam_id"], resp["provider"])
	}
}

func TestGetSession_NotFound(t *testing.T) {
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) { return nil, session.ErrNotFound },
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/sessions/missing", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestDeleteSession_Success(t *testing.T) {
	called := false
	mgr := &fakeManager{
		destroyFn: func(_ context.Context, id string, in session.StartInput) error {
			called = true
			if id != "abc" {
				t.Errorf("id = %q", id)
			}
			if in.Provider != "aws" || in.AWS == nil || in.AWS.RoleARN != "arn:aws:iam::1:role/r" {
				t.Errorf("input not forwarded: %+v", in)
			}
			return nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"provider":"aws","aws":{"role_arn":"arn:aws:iam::1:role/r"}}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodDelete, "/v1/sessions/abc", body))

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rr.Code)
	}
	if !called {
		t.Error("Destroy not called")
	}
}

func TestDeleteSession_AcceptsEmptyBody(t *testing.T) {
	mgr := &fakeManager{
		destroyFn: func(context.Context, string, session.StartInput) error { return nil },
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodDelete, "/v1/sessions/abc", nil))

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rr.Code)
	}
}

func TestDeleteSession_ManagerErrorIs500(t *testing.T) {
	mgr := &fakeManager{
		destroyFn: func(context.Context, string, session.StartInput) error {
			return errors.New("boom")
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodDelete, "/v1/sessions/abc", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

func TestValidateTask_NotReadyIs409(t *testing.T) {
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) {
			return &session.Session{ID: "abc", ExamID: "cka/x", Status: session.StatusProvisioning}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions/abc/tasks/t1/validate", nil))

	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rr.Code)
	}
}

func TestValidateTask_ExamGoneIs500(t *testing.T) {
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) {
			return &session.Session{ID: "abc", ExamID: "missing", Status: session.StatusReady}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions/abc/tasks/t1/validate", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

func TestValidateTask_TaskNotFoundIs404(t *testing.T) {
	cat := writeFixture(t)
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) {
			return &session.Session{ID: "abc", ExamID: "cka/example", Status: session.StatusReady, Kubeconfig: []byte("k")}, nil
		},
	}
	srv := newTestServer(t, cat, mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions/abc/tasks/missing-task/validate", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestValidateTask_SessionNotFoundIs404(t *testing.T) {
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) { return nil, session.ErrNotFound },
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions/abc/tasks/t1/validate", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}
