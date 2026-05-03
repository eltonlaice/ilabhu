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
	startFn   func(ctx context.Context, labID string, creds session.CloudCredentials) (*session.Session, error)
	getFn     func(id string) (*session.Session, error)
	destroyFn func(ctx context.Context, id string, creds session.CloudCredentials) error
}

func (f *fakeManager) Start(ctx context.Context, labID string, creds session.CloudCredentials) (*session.Session, error) {
	return f.startFn(ctx, labID, creds)
}
func (f *fakeManager) Get(id string) (*session.Session, error) { return f.getFn(id) }
func (f *fakeManager) Destroy(ctx context.Context, id string, creds session.CloudCredentials) error {
	return f.destroyFn(ctx, id, creds)
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

func TestListLabs_ReflectsCatalog(t *testing.T) {
	root := t.TempDir()
	labDir := filepath.Join(root, "cka", "example")
	if err := os.MkdirAll(labDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `schema_version: 1
id: cka/example
version: 1
exam: CKA
exam_objective: workloads
title: Example
summary: A short summary.
difficulty: easy
estimated_minutes: 7
infrastructure:
  provider: aws
  module: ./terraform
access:
  kind: kubeconfig
  output: kubeconfig
instructions: do it.
tasks: []
`
	if err := os.WriteFile(filepath.Join(labDir, "lab.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Load(root)
	if err != nil {
		t.Fatal(err)
	}

	srv := newTestServer(t, cat, &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/labs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var labs []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &labs); err != nil {
		t.Fatal(err)
	}
	if len(labs) != 1 {
		t.Fatalf("got %d labs, want 1", len(labs))
	}
	if labs[0]["id"] != "cka/example" {
		t.Errorf("id = %v, want cka/example", labs[0]["id"])
	}
	if labs[0]["estimated_minutes"].(float64) != 7 {
		t.Errorf("estimated_minutes = %v, want 7", labs[0]["estimated_minutes"])
	}
}

func TestGetLab_Success(t *testing.T) {
	root := t.TempDir()
	labDir := filepath.Join(root, "cka", "example")
	if err := os.MkdirAll(labDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `schema_version: 1
id: cka/example
version: 3
exam: CKA
exam_objective: workloads
title: Example
summary: x
difficulty: easy
estimated_minutes: 7
infrastructure:
  provider: aws
  module: ./terraform
  ttl_minutes: 90
access:
  kind: kubeconfig
  output: kubeconfig
instructions: do it.
tasks:
  - id: t1
    title: First task
    instructions: First instructions.
    validations: []
`
	if err := os.WriteFile(filepath.Join(labDir, "lab.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, cat, &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/labs/cka/example", nil))
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
	if first["id"] != "t1" || first["title"] != "First task" {
		t.Errorf("task fields wrong: %v", first)
	}
	infra := resp["infrastructure"].(map[string]any)
	if infra["provider"] != "aws" || infra["ttl_minutes"].(float64) != 90 {
		t.Errorf("infrastructure fields wrong: %v", infra)
	}
}

func TestGetLab_NotFound(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/labs/cka/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestCreateSession_Success(t *testing.T) {
	called := false
	mgr := &fakeManager{
		startFn: func(_ context.Context, labID string, creds session.CloudCredentials) (*session.Session, error) {
			called = true
			if labID != "cka/example" || creds.AWSRoleARN != "arn:aws:iam::1:role/r" {
				t.Errorf("unexpected args: %q %+v", labID, creds)
			}
			return &session.Session{
				ID:        "sess-123",
				LabID:     labID,
				Status:    session.StatusProvisioning,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"lab_id":"cka/example","aws_role_arn":"arn:aws:iam::1:role/r","aws_external_id":"xid"}`)
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
	if sess.ID != "sess-123" {
		t.Errorf("id = %q", sess.ID)
	}
}

func TestCreateSession_RejectsMissingLabID(t *testing.T) {
	srv := newTestServer(t, emptyCatalog(t), &fakeManager{})

	body := strings.NewReader(`{"aws_role_arn":"r"}`)
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
		startFn: func(context.Context, string, session.CloudCredentials) (*session.Session, error) {
			return nil, errors.New("aws_role_arn is required")
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"lab_id":"cka/example"}`)
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
				LabID:      "cka/example",
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
		destroyFn: func(_ context.Context, id string, creds session.CloudCredentials) error {
			called = true
			if id != "abc" {
				t.Errorf("id = %q", id)
			}
			if creds.AWSRoleARN != "arn:aws:iam::1:role/r" {
				t.Errorf("creds not forwarded: %+v", creds)
			}
			return nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	body := strings.NewReader(`{"aws_role_arn":"arn:aws:iam::1:role/r"}`)
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
		destroyFn: func(context.Context, string, session.CloudCredentials) error { return nil },
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
		destroyFn: func(context.Context, string, session.CloudCredentials) error {
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
			return &session.Session{ID: "abc", LabID: "cka/x", Status: session.StatusProvisioning}, nil
		},
	}
	srv := newTestServer(t, emptyCatalog(t), mgr)

	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sessions/abc/tasks/t1/validate", nil))

	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rr.Code)
	}
}

func TestValidateTask_LabGoneIs500(t *testing.T) {
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) {
			return &session.Session{ID: "abc", LabID: "missing", Status: session.StatusReady}, nil
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
	root := t.TempDir()
	labDir := filepath.Join(root, "cka", "example")
	_ = os.MkdirAll(labDir, 0o755)
	manifest := `schema_version: 1
id: cka/example
version: 1
exam: CKA
exam_objective: workloads
title: Example
summary: x
difficulty: easy
estimated_minutes: 1
infrastructure:
  provider: aws
  module: ./terraform
access:
  kind: kubeconfig
  output: kubeconfig
instructions: x
tasks:
  - id: real-task
    title: do it
    instructions: do it
    validations: []
`
	if err := os.WriteFile(filepath.Join(labDir, "lab.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	mgr := &fakeManager{
		getFn: func(string) (*session.Session, error) {
			return &session.Session{ID: "abc", LabID: "cka/example", Status: session.StatusReady, Kubeconfig: []byte("k")}, nil
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
