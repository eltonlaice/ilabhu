package validator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
)

// stubKubectlOnPATH writes a fake `kubectl` shell script into a temp dir and
// prepends that dir to PATH for the duration of the test. The script echoes
// $FAKE_OUT and exits with $FAKE_EXIT so each test case can drive behaviour.
func stubKubectlOnPATH(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("validator tests use a /bin/sh stub; not portable to Windows")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
if [ -n "$FAKE_OUT" ]; then printf '%s' "$FAKE_OUT"; fi
exit ${FAKE_EXIT:-0}
`
	path := filepath.Join(dir, "kubectl")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake kubectl: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// stubSSHOnPATH writes a fake `ssh` that drains stdin (the script body) and
// behaves according to FAKE_SSH_OUT and FAKE_SSH_EXIT. Same PATH prepend
// pattern as stubKubectlOnPATH.
func stubSSHOnPATH(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("validator tests use a /bin/sh stub; not portable to Windows")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
# drain the piped-in script so the caller doesn't block on a write
cat >/dev/null
if [ -n "$FAKE_SSH_OUT" ]; then printf '%s' "$FAKE_SSH_OUT"; fi
exit ${FAKE_SSH_EXIT:-0}
`
	path := filepath.Join(dir, "ssh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ssh: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func ptrString(s string) *string { return &s }
func ptrInt(i int) *int          { return &i }

func kubeAccess(b []byte) Access { return Access{Kubeconfig: b} }
func sshAccess() Access {
	return Access{SSHKeyPath: "/dev/null", SSHUser: "ubuntu", SSHHost: "1.2.3.4"}
}

func TestRun_ExpectEqualsPasses(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "Running")
	t.Setenv("FAKE_EXIT", "0")

	task := catalog.Task{
		ID: "t1",
		Validations: []catalog.Validation{
			{Kind: "kubectl", Args: []string{"get", "pod"}, ExpectEquals: ptrString("Running")},
		},
	}
	results, err := Run(context.Background(), task, kubeAccess([]byte("dummy-kubeconfig")))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected one passing result, got %+v", results)
	}
}

func TestRun_ExpectEqualsFailsOnMismatch(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "Pending")
	t.Setenv("FAKE_EXIT", "0")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectEquals: ptrString("Running")},
		},
	}
	results, err := Run(context.Background(), task, kubeAccess([]byte("k")))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].Passed {
		t.Errorf("expected failure on mismatch; message=%q", results[0].Message)
	}
}

func TestRun_ExpectContainsPasses(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "NAME    READY   STATUS    RESTARTS   AGE\nnginx   1/1     Running   0          5s")
	t.Setenv("FAKE_EXIT", "0")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectContains: ptrString("Running")},
		},
	}
	results, _ := Run(context.Background(), task, kubeAccess([]byte("k")))
	if !results[0].Passed {
		t.Errorf("expect_contains should match: %+v", results)
	}
}

func TestRun_ExpectExitCodeMatch(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "")
	t.Setenv("FAKE_EXIT", "1")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectExitCode: ptrInt(1)},
		},
	}
	results, _ := Run(context.Background(), task, kubeAccess([]byte("k")))
	if !results[0].Passed {
		t.Errorf("expect_exit_code=1 should pass when kubectl exits 1: %+v", results)
	}
}

func TestRun_ExpectExitCodeMismatch(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_EXIT", "2")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectExitCode: ptrInt(0)},
		},
	}
	results, _ := Run(context.Background(), task, kubeAccess([]byte("k")))
	if results[0].Passed {
		t.Error("expected failure on exit-code mismatch")
	}
}

func TestRun_KubectlFailureWithoutExpectExitCode(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "boom")
	t.Setenv("FAKE_EXIT", "1")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectEquals: ptrString("anything")},
		},
	}
	results, _ := Run(context.Background(), task, kubeAccess([]byte("k")))
	if results[0].Passed {
		t.Error("non-zero exit without expect_exit_code should fail")
	}
}

func TestRun_UnknownValidationKind(t *testing.T) {
	stubKubectlOnPATH(t)
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "future-kind"},
		},
	}
	results, err := Run(context.Background(), task, kubeAccess([]byte("k")))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].Passed {
		t.Error("unimplemented kind should not pass")
	}
	if results[0].Message == "" {
		t.Error("unimplemented kind should explain itself in Message")
	}
}

func TestRun_RunsAllValidationsEvenAfterFailure(t *testing.T) {
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "wrong")
	t.Setenv("FAKE_EXIT", "0")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectEquals: ptrString("right")},
			{Kind: "kubectl", ExpectContains: ptrString("wrong")},
			{Kind: "future-kind"},
		},
	}
	results, _ := Run(context.Background(), task, kubeAccess([]byte("k")))
	if len(results) != 3 {
		t.Fatalf("results len = %d, want 3", len(results))
	}
	if results[0].Passed {
		t.Error("[0] should fail")
	}
	if !results[1].Passed {
		t.Error("[1] should pass")
	}
	if results[2].Passed {
		t.Error("[2] (unimplemented) should fail")
	}
}

func TestRun_KubectlTaskWithoutKubeconfigErrors(t *testing.T) {
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectEquals: ptrString("x")},
		},
	}
	if _, err := Run(context.Background(), task, Access{}); err == nil {
		t.Error("expected error when a kubectl task has no kubeconfig")
	}
}

func TestRun_ShellTaskWithoutKubeconfigStillRuns(t *testing.T) {
	// A task that only has shell validations should not require a kubeconfig.
	stubSSHOnPATH(t)
	t.Setenv("FAKE_SSH_EXIT", "0")
	t.Setenv("FAKE_SSH_OUT", "ok")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "echo ok"},
		},
	}
	results, err := Run(context.Background(), task, sshAccess())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !results[0].Passed {
		t.Errorf("shell-only task should pass without a kubeconfig: %+v", results)
	}
}

// --- shell validation kind ---

func TestRun_Shell_DefaultExitCodeZeroPasses(t *testing.T) {
	stubSSHOnPATH(t)
	t.Setenv("FAKE_SSH_EXIT", "0")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "/bin/true"},
		},
	}
	results, _ := Run(context.Background(), task, sshAccess())
	if !results[0].Passed {
		t.Errorf("default-exit-0 should pass: %+v", results)
	}
}

func TestRun_Shell_DefaultExitNonZeroFails(t *testing.T) {
	stubSSHOnPATH(t)
	t.Setenv("FAKE_SSH_EXIT", "1")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "/bin/false"},
		},
	}
	results, _ := Run(context.Background(), task, sshAccess())
	if results[0].Passed {
		t.Error("non-zero exit should fail without expect_exit_code override")
	}
}

func TestRun_Shell_ExpectExitCodeMatch(t *testing.T) {
	stubSSHOnPATH(t)
	t.Setenv("FAKE_SSH_EXIT", "2")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "exit 2", ExpectExitCode: ptrInt(2)},
		},
	}
	results, _ := Run(context.Background(), task, sshAccess())
	if !results[0].Passed {
		t.Errorf("expect_exit_code=2 should pass: %+v", results)
	}
}

func TestRun_Shell_ExpectEqualsTrimsAndMatches(t *testing.T) {
	stubSSHOnPATH(t)
	t.Setenv("FAKE_SSH_EXIT", "0")
	t.Setenv("FAKE_SSH_OUT", "  Running\n")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "echo Running", ExpectEquals: ptrString("Running")},
		},
	}
	results, _ := Run(context.Background(), task, sshAccess())
	if !results[0].Passed {
		t.Errorf("trimmed expect_equals should pass: %+v", results)
	}
}

func TestRun_Shell_ExpectContains(t *testing.T) {
	stubSSHOnPATH(t)
	t.Setenv("FAKE_SSH_EXIT", "0")
	t.Setenv("FAKE_SSH_OUT", "ubuntu 24.04.1 LTS")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "lsb_release -d", ExpectContains: ptrString("24.04")},
		},
	}
	results, _ := Run(context.Background(), task, sshAccess())
	if !results[0].Passed {
		t.Errorf("expect_contains should match substring: %+v", results)
	}
}

func TestRun_Shell_EmptyScriptFails(t *testing.T) {
	stubSSHOnPATH(t)
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: ""},
		},
	}
	results, _ := Run(context.Background(), task, sshAccess())
	if results[0].Passed {
		t.Error("empty script should fail validation")
	}
}

// --- http validation kind ---

func TestRun_HTTP_DefaultStatus200Passes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: srv.URL},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if !results[0].Passed {
		t.Errorf("default 200 should pass: %+v", results)
	}
}

func TestRun_HTTP_ExpectStatusOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()
	teapot := http.StatusTeapot

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: srv.URL, ExpectStatus: &teapot},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if !results[0].Passed {
		t.Errorf("expect_status=418 should pass when server returns 418: %+v", results)
	}
}

func TestRun_HTTP_StatusMismatchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: srv.URL},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if results[0].Passed {
		t.Error("server 500 should fail default-200 expectation")
	}
}

func TestRun_HTTP_ExpectBodyContains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","version":"1.2.3"}`))
	}))
	defer srv.Close()

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: srv.URL, ExpectBodyHas: ptrString(`"version":"1.2.3"`)},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if !results[0].Passed {
		t.Errorf("expect_body_contains should match: %+v", results)
	}
}

func TestRun_HTTP_ExpectBodyContainsMissingFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: srv.URL, ExpectBodyHas: ptrString("ok")},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if results[0].Passed {
		t.Error("expect_body_contains absent should fail")
	}
}

func TestRun_HTTP_PublicIPSubstitution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// httptest.Server URL is like http://127.0.0.1:54321. Split it so we can
	// rebuild it with the {{public_ip}} substitution; access.SSHHost becomes
	// the host:port substring.
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: u.Scheme + "://{{public_ip}}/healthz"},
		},
	}
	access := Access{SSHHost: u.Host}
	results, _ := Run(context.Background(), task, access)
	if !results[0].Passed {
		t.Errorf("{{public_ip}} substitution should reach the test server: %+v", results)
	}
}

func TestRun_HTTP_EmptyURLFails(t *testing.T) {
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: ""},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if results[0].Passed {
		t.Error("empty url should fail validation")
	}
}

func TestRun_HTTP_RequestErrorFails(t *testing.T) {
	// 127.0.0.1:1 is reliably refused on the test host.
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "http", URL: "http://127.0.0.1:1/"},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if results[0].Passed {
		t.Error("unreachable url should fail validation, not error")
	}
}

func TestRun_Shell_MissingAccessFails(t *testing.T) {
	stubSSHOnPATH(t)
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "echo ok"},
		},
	}
	results, _ := Run(context.Background(), task, Access{})
	if results[0].Passed {
		t.Error("missing SSH access should fail shell validation, not error")
	}
}
