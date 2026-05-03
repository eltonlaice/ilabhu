package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/eltonlaice/ilabhu/control-plane/internal/session"
	"github.com/eltonlaice/ilabhu/control-plane/internal/validator"
)

type createSessionRequest struct {
	LabID         string `json:"lab_id"`
	AWSRoleARN    string `json:"aws_role_arn"`
	AWSExternalID string `json:"aws_external_id"`
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.LabID == "" {
		writeError(w, http.StatusBadRequest, "lab_id is required")
		return
	}
	sess, err := s.Manager.Start(r.Context(), req.LabID, session.CloudCredentials{
		AWSRoleARN:    req.AWSRoleARN,
		AWSExternalID: req.AWSExternalID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, sess)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.Manager.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	resp := map[string]any{
		"id":         sess.ID,
		"lab_id":     sess.LabID,
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
	var req createSessionRequest
	body, _ := io.ReadAll(r.Body)
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
			return
		}
	}
	if err := s.Manager.Destroy(r.Context(), id, session.CloudCredentials{
		AWSRoleARN:    req.AWSRoleARN,
		AWSExternalID: req.AWSExternalID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleValidateTask(w http.ResponseWriter, r *http.Request) {
	sessID := r.PathValue("id")
	taskID := r.PathValue("task_id")

	sess, err := s.Manager.Store.Get(sessID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if sess.Status != session.StatusReady {
		writeError(w, http.StatusConflict, "session is not ready (status: "+string(sess.Status)+")")
		return
	}
	lab, ok := s.Catalog.Get(sess.LabID)
	if !ok {
		writeError(w, http.StatusInternalServerError, "lab no longer in catalog")
		return
	}
	taskIdx := -1
	for i, t := range lab.Tasks {
		if t.ID == taskID {
			taskIdx = i
			break
		}
	}
	if taskIdx < 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	results, err := validator.Run(r.Context(), lab.Tasks[taskIdx], sess.Kubeconfig)
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
