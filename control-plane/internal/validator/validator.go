package validator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
)

// matchRegex compiles `pattern` and returns whether it matches `got`. A bad
// pattern is reported in the message so the exam author finds out at run-time
// instead of guessing why their validation never passes.
func matchRegex(got, pattern string) (bool, string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Sprintf("invalid expect_regex %q: %v", pattern, err)
	}
	if !re.MatchString(got) {
		return false, fmt.Sprintf("expected output to match /%s/, got %q", pattern, got)
	}
	return true, ""
}

// Access carries everything a validation kind might need to reach the
// session's environment. Each kind reads the subset it needs; missing fields
// produce a clear per-validation failure rather than a fatal error.
type Access struct {
	// Kubeconfig is the contents of the session's kubeconfig file. Required
	// for kind: kubectl.
	Kubeconfig []byte
	// SSHKeyPath, SSHUser and SSHHost describe how to reach the exam VM
	// (for single-node exams, the only host; for multi-node, the access host
	// reported by the Terraform module). Required for kind: shell.
	SSHKeyPath string
	SSHUser    string
	SSHHost    string
}

// Result is the outcome of one validation rule.
type Result struct {
	Index   int    `json:"index"`
	Kind    string `json:"kind"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// Run evaluates each validation in `task` in order. Validations run regardless
// of earlier failures so the caller sees the full picture.
func Run(ctx context.Context, task catalog.Task, access Access) ([]Result, error) {
	var kubePath string
	var cleanup func()
	if hasKubectl(task) {
		var err error
		kubePath, cleanup, err = writeKubeconfig(access.Kubeconfig)
		if err != nil {
			return nil, err
		}
		defer cleanup()
	}

	results := make([]Result, 0, len(task.Validations))
	for i, v := range task.Validations {
		r := Result{Index: i, Kind: v.Kind}
		switch v.Kind {
		case "kubectl":
			r.Passed, r.Message = runKubectl(ctx, kubePath, v)
		case "shell":
			r.Passed, r.Message = runShell(ctx, access, v)
		case "http":
			r.Passed, r.Message = runHTTP(ctx, access, v, httpClient)
		default:
			r.Passed = false
			r.Message = fmt.Sprintf("validation kind %q not implemented", v.Kind)
		}
		results = append(results, r)
	}
	return results, nil
}

func hasKubectl(task catalog.Task) bool {
	for _, v := range task.Validations {
		if v.Kind == "kubectl" {
			return true
		}
	}
	return false
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
	if v.ExpectRegex != nil {
		if ok, msg := matchRegex(got, *v.ExpectRegex); !ok {
			return false, msg
		}
	}
	return true, ""
}

// runShell pipes v.Script over SSH to /bin/sh on the exam host. Default pass
// condition is exit code 0; expect_exit_code overrides; expect_contains and
// expect_equals match against the captured stdout+stderr.
func runShell(ctx context.Context, access Access, v catalog.Validation) (bool, string) {
	if v.Script == "" {
		return false, "shell validation requires a non-empty script"
	}
	if access.SSHKeyPath == "" || access.SSHUser == "" || access.SSHHost == "" {
		return false, "session has no SSH access for shell validations"
	}

	cmd := exec.CommandContext(ctx, "ssh",
		"-i", access.SSHKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		access.SSHUser+"@"+access.SSHHost,
		"/bin/sh", "-s",
	)
	cmd.Stdin = strings.NewReader(v.Script)
	out, err := cmd.CombinedOutput()
	got := strings.TrimSpace(string(out))

	expectExit := 0
	if v.ExpectExitCode != nil {
		expectExit = *v.ExpectExitCode
	}
	actualExit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			actualExit = ee.ExitCode()
		} else {
			return false, fmt.Sprintf("ssh run error: %v", err)
		}
	}
	if actualExit != expectExit {
		return false, fmt.Sprintf("expected exit code %d, got %d (%s)", expectExit, actualExit, got)
	}
	if v.ExpectEquals != nil && got != *v.ExpectEquals {
		return false, fmt.Sprintf("expected %q, got %q", *v.ExpectEquals, got)
	}
	if v.ExpectContains != nil && !strings.Contains(got, *v.ExpectContains) {
		return false, fmt.Sprintf("expected output to contain %q, got %q", *v.ExpectContains, got)
	}
	if v.ExpectRegex != nil {
		if ok, msg := matchRegex(got, *v.ExpectRegex); !ok {
			return false, msg
		}
	}
	return true, ""
}

// httpClient is shared across runHTTP calls. Timeout is intentionally generous
// because a freshly-provisioned lab service may take a few seconds to wire
// up; per-validation context still bounds the request.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// runHTTP issues a GET against v.URL. The URL supports a `{{public_ip}}`
// substitution that resolves to access.SSHHost, so manifests can target a
// NodePort or LoadBalancer endpoint without hard-coding addresses.
//
// Default pass condition is status 200; expect_status overrides;
// expect_body_contains matches against the (size-capped) response body.
func runHTTP(ctx context.Context, access Access, v catalog.Validation, client *http.Client) (bool, string) {
	if v.URL == "" {
		return false, "http validation requires a non-empty url"
	}
	url := strings.ReplaceAll(v.URL, "{{public_ip}}", access.SSHHost)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Sprintf("invalid url: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("http request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	wantStatus := http.StatusOK
	if v.ExpectStatus != nil {
		wantStatus = *v.ExpectStatus
	}
	if resp.StatusCode != wantStatus {
		return false, fmt.Sprintf("expected status %d, got %d", wantStatus, resp.StatusCode)
	}

	if v.ExpectBodyHas != nil || v.ExpectRegex != nil {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return false, fmt.Sprintf("read body: %v", err)
		}
		if v.ExpectBodyHas != nil && !strings.Contains(string(body), *v.ExpectBodyHas) {
			return false, fmt.Sprintf("expected body to contain %q", *v.ExpectBodyHas)
		}
		if v.ExpectRegex != nil {
			if ok, msg := matchRegex(string(body), *v.ExpectRegex); !ok {
				return false, msg
			}
		}
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
