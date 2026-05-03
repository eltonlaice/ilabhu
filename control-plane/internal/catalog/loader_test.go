package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validManifest = `schema_version: 1
id: cka/example
version: 1
exam: CKA
exam_objective: Workloads
title: Example
summary: An example lab.
difficulty: easy
estimated_minutes: 5
infrastructure:
  provider: aws
  module: ./terraform
  ttl_minutes: 60
  inputs:
    instance_type: t3.small
access:
  kind: kubeconfig
  output: kubeconfig
instructions: |
  Do the thing.
tasks:
  - id: do-it
    title: Do it
    instructions: Do it.
    validations:
      - kind: kubectl
        args: [get, pods]
        expect_contains: Running
`

func writeManifest(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "lab.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestParseManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, validManifest)

	m, err := parseManifest(path)
	if err != nil {
		t.Fatalf("parseManifest: %v", err)
	}
	if m.ID != "cka/example" {
		t.Errorf("ID = %q, want cka/example", m.ID)
	}
	if m.Dir != dir {
		t.Errorf("Dir = %q, want %q", m.Dir, dir)
	}
	if len(m.Tasks) != 1 {
		t.Fatalf("Tasks = %d, want 1", len(m.Tasks))
	}
	if len(m.Tasks[0].Validations) != 1 {
		t.Fatalf("Validations = %d, want 1", len(m.Tasks[0].Validations))
	}
	v := m.Tasks[0].Validations[0]
	if v.Kind != "kubectl" {
		t.Errorf("validation kind = %q, want kubectl", v.Kind)
	}
	if v.ExpectContains == nil || *v.ExpectContains != "Running" {
		t.Errorf("expect_contains = %v, want Running", v.ExpectContains)
	}
}

func TestParseManifest_RejectsUnsupportedSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	bad := strings.Replace(validManifest, "schema_version: 1", "schema_version: 99", 1)
	path := writeManifest(t, dir, bad)

	_, err := parseManifest(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Errorf("error = %q, want it to mention schema_version", err)
	}
}

func TestParseManifest_RejectsMissingID(t *testing.T) {
	dir := t.TempDir()
	bad := strings.Replace(validManifest, "id: cka/example\n", "", 1)
	path := writeManifest(t, dir, bad)

	_, err := parseManifest(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "id") {
		t.Errorf("error = %q, want it to mention id", err)
	}
}

func TestParseManifest_RejectsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, "not: valid: yaml: : :")

	if _, err := parseManifest(path); err == nil {
		t.Fatal("expected error on invalid yaml, got nil")
	}
}

func TestLoad_WalksTree(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "cka", "one"), validManifest)
	twoManifest := strings.Replace(validManifest, "id: cka/example", "id: cka/two", 1)
	writeManifest(t, filepath.Join(root, "cka", "two"), twoManifest)

	cat, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := len(cat.List()); got != 2 {
		t.Fatalf("List len = %d, want 2", got)
	}
	if _, ok := cat.Get("cka/example"); !ok {
		t.Error("expected to find cka/example")
	}
	if _, ok := cat.Get("cka/two"); !ok {
		t.Error("expected to find cka/two")
	}
	if _, ok := cat.Get("does/not-exist"); ok {
		t.Error("expected not to find missing id")
	}
}

func TestLoad_DetectsDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "a"), validManifest)
	writeManifest(t, filepath.Join(root, "b"), validManifest)

	_, err := Load(root)
	if err == nil {
		t.Fatal("expected duplicate error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error = %q, want it to mention duplicate", err)
	}
}

func TestLoad_MissingRootIsEmpty(t *testing.T) {
	cat, err := Load(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := len(cat.List()); got != 0 {
		t.Errorf("List len = %d, want 0", got)
	}
}

func TestLoad_IgnoresUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "cka", "example"), validManifest)
	if err := os.WriteFile(filepath.Join(root, "cka", "example", "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cka", "example", "other.yaml"), []byte("x: 1"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := len(cat.List()); got != 1 {
		t.Errorf("List len = %d, want 1", got)
	}
}
