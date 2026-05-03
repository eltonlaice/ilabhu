package validator

import (
	"context"
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

func ptrString(s string) *string { return &s }
func ptrInt(i int) *int          { return &i }

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
	results, err := Run(context.Background(), task, []byte("dummy-kubeconfig"))
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
	results, err := Run(context.Background(), task, []byte("k"))
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
	results, _ := Run(context.Background(), task, []byte("k"))
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
	results, _ := Run(context.Background(), task, []byte("k"))
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
	results, _ := Run(context.Background(), task, []byte("k"))
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
	results, _ := Run(context.Background(), task, []byte("k"))
	if results[0].Passed {
		t.Error("non-zero exit without expect_exit_code should fail")
	}
}

func TestRun_UnknownValidationKind(t *testing.T) {
	stubKubectlOnPATH(t)
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "shell", Script: "echo hi"},
		},
	}
	results, err := Run(context.Background(), task, []byte("k"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].Passed {
		t.Error("unimplemented kind should not pass")
	}
}

func TestRun_RunsAllValidationsEvenAfterFailure(t *testing.T) {
	// Each validation invokes kubectl with the same env, so they share
	// behaviour. The point of this test is that .Run keeps going past a
	// failure and reports per-index results.
	stubKubectlOnPATH(t)
	t.Setenv("FAKE_OUT", "wrong")
	t.Setenv("FAKE_EXIT", "0")

	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectEquals: ptrString("right")},
			{Kind: "kubectl", ExpectContains: ptrString("wrong")},
			{Kind: "shell", Script: "noop"},
		},
	}
	results, _ := Run(context.Background(), task, []byte("k"))
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
		t.Error("[2] (shell, unimplemented) should fail")
	}
}

func TestRun_EmptyKubeconfigErrors(t *testing.T) {
	task := catalog.Task{
		Validations: []catalog.Validation{
			{Kind: "kubectl", ExpectEquals: ptrString("x")},
		},
	}
	if _, err := Run(context.Background(), task, nil); err == nil {
		t.Error("expected error on empty kubeconfig")
	}
}
