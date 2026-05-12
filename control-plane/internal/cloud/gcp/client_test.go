package gcp

import (
	"strings"
	"testing"
)

const validSAKey = `{
  "type": "service_account",
  "project_id": "ilabhu-demo-12345",
  "private_key_id": "abc",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQ...\n-----END PRIVATE KEY-----\n",
  "client_email": "ilabhu-runner@ilabhu-demo-12345.iam.gserviceaccount.com",
  "client_id": "1234567890",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token"
}`

func TestNewCredentials_ParsesProject(t *testing.T) {
	c, err := NewCredentials(validSAKey)
	if err != nil {
		t.Fatalf("NewCredentials: %v", err)
	}
	if c.Project != "ilabhu-demo-12345" {
		t.Errorf("Project = %q, want ilabhu-demo-12345", c.Project)
	}
	if c.ServiceAccountKey != validSAKey {
		t.Error("ServiceAccountKey should round-trip verbatim")
	}
}

func TestNewCredentials_EmptyRejected(t *testing.T) {
	_, err := NewCredentials("")
	if err == nil {
		t.Fatal("expected error on empty SA key")
	}
}

func TestNewCredentials_MalformedJSONRejected(t *testing.T) {
	_, err := NewCredentials("{not valid json")
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
	if !strings.Contains(err.Error(), "invalid service account key JSON") {
		t.Errorf("error = %q, want it to mention invalid JSON", err)
	}
}

func TestNewCredentials_MissingProjectRejected(t *testing.T) {
	_, err := NewCredentials(`{"type":"service_account","client_email":"x@y.iam.gserviceaccount.com"}`)
	if err == nil {
		t.Fatal("expected error when project_id is missing")
	}
	if !strings.Contains(err.Error(), "project_id") {
		t.Errorf("error = %q, want it to mention project_id", err)
	}
}

func TestNewCredentials_RejectsWrongType(t *testing.T) {
	_, err := NewCredentials(`{"type":"user","project_id":"x"}`)
	if err == nil {
		t.Fatal("expected error on non-service_account type")
	}
}

func TestCredentials_AsEnv(t *testing.T) {
	c, err := NewCredentials(validSAKey)
	if err != nil {
		t.Fatal(err)
	}
	env := c.AsEnv()
	if len(env) != 2 {
		t.Fatalf("env len = %d, want 2 (got %v)", len(env), env)
	}

	var sawCreds, sawProject bool
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "GOOGLE_CREDENTIALS="):
			sawCreds = true
			if !strings.Contains(e, "ilabhu-demo-12345") {
				t.Error("GOOGLE_CREDENTIALS should carry the JSON verbatim")
			}
		case e == "TF_VAR_project=ilabhu-demo-12345":
			sawProject = true
		default:
			t.Errorf("unexpected env entry: %q", e)
		}
	}
	if !sawCreds {
		t.Error("missing GOOGLE_CREDENTIALS")
	}
	if !sawProject {
		t.Error("missing TF_VAR_project")
	}
}

func TestCredentials_AsEnvDoesNotLeakProjectOutsideEnvVar(t *testing.T) {
	// Sanity: project id should not appear as an env key.
	c := Credentials{ServiceAccountKey: validSAKey, Project: "secret-project"}
	for _, e := range c.AsEnv() {
		parts := strings.SplitN(e, "=", 2)
		if parts[0] == "secret-project" {
			t.Errorf("project id leaked into env key: %q", e)
		}
	}
}
