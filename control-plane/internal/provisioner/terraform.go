package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Terraform shells out to the `terraform` binary. It does not link the HCL
// library because it only needs to drive a module; this also ensures behaviour
// matches what self-hosters get from their installed toolchain.
type Terraform struct {
	// Binary is the name or path of the terraform executable. Defaults to
	// "terraform" if empty.
	Binary string

	// Stdout and Stderr receive the terraform process output. If nil they are
	// discarded.
	Stdout io.Writer
	Stderr io.Writer
}

// Apply runs `terraform init` and `terraform apply -auto-approve` in workDir
// with the given variables and environment. workDir must already contain a
// copy of the lab's terraform module plus a `terraform.tfvars.json` file
// generated from `vars`.
func (t *Terraform) Apply(ctx context.Context, workDir string, vars map[string]any, env []string) error {
	if err := writeVarsFile(workDir, vars); err != nil {
		return err
	}
	if err := t.run(ctx, workDir, env, "init", "-input=false", "-no-color"); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	if err := t.run(ctx, workDir, env, "apply", "-auto-approve", "-input=false", "-no-color"); err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}
	return nil
}

// Destroy runs `terraform destroy -auto-approve` in workDir.
func (t *Terraform) Destroy(ctx context.Context, workDir string, env []string) error {
	if err := t.run(ctx, workDir, env, "destroy", "-auto-approve", "-input=false", "-no-color"); err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}
	return nil
}

// Outputs returns the parsed JSON output of `terraform output -json`.
func (t *Terraform) Outputs(ctx context.Context, workDir string, env []string) (map[string]Output, error) {
	cmd := exec.CommandContext(ctx, t.binary(), "output", "-json")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), env...)
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform output: %w", err)
	}
	out := map[string]Output{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse terraform output: %w", err)
	}
	return out, nil
}

// Output is one entry of `terraform output -json`.
type Output struct {
	Value     any  `json:"value"`
	Sensitive bool `json:"sensitive"`
}

func (t *Terraform) run(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, t.binary(), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = t.Stdout
	cmd.Stderr = t.Stderr
	return cmd.Run()
}

func (t *Terraform) binary() string {
	if t.Binary != "" {
		return t.Binary
	}
	return "terraform"
}

func writeVarsFile(dir string, vars map[string]any) error {
	if vars == nil {
		vars = map[string]any{}
	}
	raw, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "terraform.tfvars.json"), raw, 0o600)
}
