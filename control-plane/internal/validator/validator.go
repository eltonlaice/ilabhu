package validator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
)

// Result is the outcome of one validation rule.
type Result struct {
	Index   int    `json:"index"`
	Kind    string `json:"kind"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// Run evaluates each validation in `task` in order. Validations run regardless
// of earlier failures so the caller sees the full picture.
func Run(ctx context.Context, task catalog.Task, kubeconfig []byte) ([]Result, error) {
	kubePath, cleanup, err := writeKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	results := make([]Result, 0, len(task.Validations))
	for i, v := range task.Validations {
		r := Result{Index: i, Kind: v.Kind}
		switch v.Kind {
		case "kubectl":
			r.Passed, r.Message = runKubectl(ctx, kubePath, v)
		default:
			r.Passed = false
			r.Message = fmt.Sprintf("validation kind %q not implemented", v.Kind)
		}
		results = append(results, r)
	}
	return results, nil
}

func runKubectl(ctx context.Context, kubeconfigPath string, v catalog.Validation) (bool, string) {
	cmd := exec.CommandContext(ctx, "kubectl", v.Args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	out, err := cmd.CombinedOutput()
	got := strings.TrimSpace(string(out))

	if v.ExpectExitCode != nil {
		exit := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exit = ee.ExitCode()
			} else {
				return false, fmt.Sprintf("kubectl run error: %v", err)
			}
		}
		if exit != *v.ExpectExitCode {
			return false, fmt.Sprintf("expected exit code %d, got %d (%s)", *v.ExpectExitCode, exit, got)
		}
		return true, ""
	}

	if err != nil {
		return false, fmt.Sprintf("kubectl failed: %v: %s", err, got)
	}
	if v.ExpectEquals != nil && got != *v.ExpectEquals {
		return false, fmt.Sprintf("expected %q, got %q", *v.ExpectEquals, got)
	}
	if v.ExpectContains != nil && !strings.Contains(got, *v.ExpectContains) {
		return false, fmt.Sprintf("expected output to contain %q, got %q", *v.ExpectContains, got)
	}
	return true, ""
}

func writeKubeconfig(data []byte) (string, func(), error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("session has no kubeconfig")
	}
	f, err := os.CreateTemp("", "ilabhu-kubeconfig-*")
	if err != nil {
		return "", nil, err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}
