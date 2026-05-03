package provisioner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareWorkdir_CopiesTreeIncludingNestedDirs(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "main.tf"), "resource \"null\" \"x\" {}\n")
	mustWrite(t, filepath.Join(src, "variables.tf"), "variable \"x\" {}\n")
	mustWrite(t, filepath.Join(src, "modules", "vpc", "main.tf"), "module {}\n")

	dst := filepath.Join(t.TempDir(), "module")
	if err := PrepareWorkdir(src, dst); err != nil {
		t.Fatalf("PrepareWorkdir: %v", err)
	}

	for _, rel := range []string{"main.tf", "variables.tf", "modules/vpc/main.tf"} {
		p := filepath.Join(dst, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s in dst, got: %v", rel, err)
		}
	}
}

func TestPrepareWorkdir_FilesAreCopiedNotLinked(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "main.tf"), "original\n")

	dst := filepath.Join(t.TempDir(), "module")
	if err := PrepareWorkdir(src, dst); err != nil {
		t.Fatalf("PrepareWorkdir: %v", err)
	}

	// Mutating dst must not change src — the per-session copy must be
	// independent so two sessions of the same lab don't share state.
	if err := os.WriteFile(filepath.Join(dst, "main.tf"), []byte("mutated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(src, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original\n" {
		t.Errorf("src changed after dst mutation: %q", got)
	}
}

func TestPrepareWorkdir_MissingSourceErrors(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "module")
	err := PrepareWorkdir(filepath.Join(t.TempDir(), "does-not-exist"), dst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
