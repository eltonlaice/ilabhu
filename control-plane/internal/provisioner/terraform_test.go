package provisioner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteVarsFile_WritesJSONAtConventionalPath(t *testing.T) {
	dir := t.TempDir()
	vars := map[string]any{
		"region":         "eu-west-1",
		"instance_type":  "t3.small",
		"session_id":     "abc-123",
		"ssh_public_key": "ssh-ed25519 AAAA...",
	}

	if err := writeVarsFile(dir, vars); err != nil {
		t.Fatalf("writeVarsFile: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "terraform.tfvars.json"))
	if err != nil {
		t.Fatalf("read tfvars: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for k, want := range vars {
		if got[k] != want {
			t.Errorf("tfvars[%q] = %v, want %v", k, got[k], want)
		}
	}
}

func TestWriteVarsFile_NilVarsWritesEmptyObject(t *testing.T) {
	dir := t.TempDir()
	if err := writeVarsFile(dir, nil); err != nil {
		t.Fatalf("writeVarsFile: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "terraform.tfvars.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty object, got %v", got)
	}
}

func TestTerraform_BinaryDefaultsToTerraform(t *testing.T) {
	tf := &Terraform{}
	if got := tf.binary(); got != "terraform" {
		t.Errorf("binary() = %q, want terraform", got)
	}

	tf2 := &Terraform{Binary: "/opt/bin/tofu"}
	if got := tf2.binary(); got != "/opt/bin/tofu" {
		t.Errorf("binary() = %q, want /opt/bin/tofu", got)
	}
}
