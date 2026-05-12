// Package gcp is the Google Cloud BYO-cloud adapter for ilabhud.
//
// Authentication is via a Service Account key in JSON form. The user pastes
// the full key on session start; ilabhud parses the project_id out of it,
// exports the JSON content via GOOGLE_CREDENTIALS and the project id via
// TF_VAR_project for the Terraform subprocess, and drops everything when
// that subprocess exits.
//
// A future revision will add Workload Identity Federation support for
// deployments that can issue OIDC tokens; for now the SA key path is what
// every self-hoster can run.
package gcp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Credentials is the per-session GCP bundle. ServiceAccountKey holds the raw
// JSON; Project is the project_id extracted from that JSON. Both travel
// through StartInput, into env vars for the Terraform subprocess, and are
// dropped when the subprocess exits.
type Credentials struct {
	ServiceAccountKey string
	Project           string
}

// NewCredentials validates the service account key JSON and extracts the
// project id from it. Returns an error if the JSON is malformed or missing
// `project_id` — both of which would otherwise surface as a confusing
// Terraform error.
func NewCredentials(serviceAccountKeyJSON string) (Credentials, error) {
	if serviceAccountKeyJSON == "" {
		return Credentials{}, errors.New("service account key is required")
	}
	var parsed struct {
		ProjectID string `json:"project_id"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal([]byte(serviceAccountKeyJSON), &parsed); err != nil {
		return Credentials{}, fmt.Errorf("invalid service account key JSON: %w", err)
	}
	if parsed.ProjectID == "" {
		return Credentials{}, errors.New("service account key is missing project_id")
	}
	if parsed.Type != "" && parsed.Type != "service_account" {
		return Credentials{}, fmt.Errorf("expected service account key, got type %q", parsed.Type)
	}
	return Credentials{
		ServiceAccountKey: serviceAccountKeyJSON,
		Project:           parsed.ProjectID,
	}, nil
}

// AsEnv returns the credentials as Terraform-compatible environment variables.
// GOOGLE_CREDENTIALS is read by the Google Terraform provider directly; the
// TF_VAR_project pair injects the project id into the exam's TF module
// without the manifest needing to hard-code it.
func (c Credentials) AsEnv() []string {
	return []string{
		"GOOGLE_CREDENTIALS=" + c.ServiceAccountKey,
		"TF_VAR_project=" + c.Project,
	}
}
