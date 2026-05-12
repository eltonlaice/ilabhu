package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/eltonlaice/ilabhu/control-plane/internal/session"
	"github.com/eltonlaice/ilabhu/control-plane/internal/validator"
)

// awsCredsBody is the AWS provider credentials block in the create/destroy
// request body.
type awsCredsBody struct {
	RoleARN    string `json:"role_arn"`
	ExternalID string `json:"external_id"`
}

// doCredsBody is the DigitalOcean provider credentials block.
type doCredsBody struct {
	Token string `json:"token"`
}

// gcpCredsBody is the Google Cloud provider credentials block. The project id
// is extracted server-side from `service_account_key`'s JSON.
type gcpCredsBody struct {
	ServiceAccountKey string `json:"service_account_key"`
}

// sessionRequest is the create- and destroy-session request body. Each
// supported provider has its own optional block; the active one is selected
// by the `provider` discriminator.
type sessionRequest struct {
	ExamID       string        `json:"exam_id,omitempty"`
	Provider     string        `json:"provider"`
	AWS          *awsCredsBody `json:"aws,omitempty"`
	DigitalOcean *doCredsBody  `json:"digitalocean,omitempty"`
	GCP          *gcpCredsBody `json:"gcp,omitempty"`
}

func (r *sessionRequest) toStartInput() session.StartInput {
	in := session.StartInput{Provider: r.Provider}
	if r.AWS != nil {
		in.AWS = &session.AWSCredentials{
			RoleARN:    r.AWS.RoleARN,
			ExternalID: r.AWS.ExternalID,
		}
	}
	if r.DigitalOcean != nil {
		in.DigitalOcean = &session.DOCredentials{
			Token: r.DigitalOcean.Token,
		}
	}
	if r.GCP != nil {
		in.GCP = &session.GCPCredentials{
			ServiceAccountKey: r.GCP.ServiceAccountKey,
		}
	}
	return in
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.ExamID == "" {
		writeError(w, http.StatusBadRequest, "exam_id is required")
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	sess, err := s.Manager.Start(r.Context(), req.ExamID, req.toStartInput())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, sess)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.Manager.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	resp := map[string]any{
		"id":         sess.ID,
		"exam_id":    sess.ExamID,
		"provider":   sess.Provider,
		"status":     sess.Status,
		"created_at": sess.CreatedAt,
		"updated_at": sess.UpdatedAt,
		"outputs":    sess.Outputs,
		"error":      sess.Error,
	}
	if len(sess.Kubeconfig) > 0 {
		resp["kubeconfig_b64"] = base64.StdEncoding.EncodeToString(sess.Kubeconfig)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req sessionRequest
	body, _ := io.ReadAll(r.Body)
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
			return
		}
	}
	if err := s.Manager.Destroy(r.Context(), id, req.toStartInput()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleValidateTask(w http.ResponseWriter, r *http.Request) {
	sessID := r.PathValue("id")
	taskID := r.PathValue("task_id")

	sess, err := s.Manager.Get(sessID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if sess.Status != session.StatusReady {
		writeError(w, http.StatusConflict, "session is not ready (status: "+string(sess.Status)+")")
		return
	}
	exam, ok := s.Catalog.Get(sess.ExamID)
	if !ok {
		writeError(w, http.StatusInternalServerError, "exam no longer in catalog")
		return
	}
	taskIdx := -1
	for i, t := range exam.Tasks {
		if t.ID == taskID {
			taskIdx = i
			break
		}
	}
	if taskIdx < 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	access := validator.Access{
		Kubeconfig: sess.Kubeconfig,
		SSHKeyPath: sess.SSHPrivateKeyPath,
		SSHUser:    outputString(sess.Outputs, "ssh_user"),
		SSHHost:    outputString(sess.Outputs, "public_ip"),
	}
	results, err := validator.Run(r.Context(), exam.Tasks[taskIdx], access)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	allPassed := true
	for _, res := range results {
		if !res.Passed {
			allPassed = false
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":    taskID,
		"all_passed": allPassed,
		"results":    results,
	})
}

// outputString returns m[key] coerced to a string, or "" if the value is
// missing or not a string. Terraform outputs are loaded as map[string]any
// so the kind assertion is the path of least surprise.
func outputString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
