package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
)

func TestValidateBYOHostsRoles_Accepts(t *testing.T) {
	spec := catalog.ProviderSpec{
		Roles: []catalog.HostRole{
			{Name: "control-plane", Count: 1},
			{Name: "worker", Count: 2},
		},
	}
	hosts := []BYOHost{
		{Role: "control-plane", Address: "10.0.0.1", SSHUser: "ubuntu"},
		{Role: "worker", Address: "10.0.0.2", SSHUser: "ubuntu"},
		{Role: "worker", Address: "10.0.0.3", SSHUser: "ubuntu"},
	}
	if err := validateBYOHostsRoles(spec, hosts); err != nil {
		t.Errorf("validateBYOHostsRoles: %v", err)
	}
}

func TestValidateBYOHostsRoles_AllowsExtraHostsInDeclaredRole(t *testing.T) {
	spec := catalog.ProviderSpec{
		Roles: []catalog.HostRole{{Name: "worker", Count: 2}},
	}
	hosts := []BYOHost{
		{Role: "worker", Address: "10.0.0.1", SSHUser: "ubuntu"},
		{Role: "worker", Address: "10.0.0.2", SSHUser: "ubuntu"},
		{Role: "worker", Address: "10.0.0.3", SSHUser: "ubuntu"},
	}
	if err := validateBYOHostsRoles(spec, hosts); err != nil {
		t.Errorf("extra hosts in a declared role should be fine: %v", err)
	}
}

func TestValidateBYOHostsRoles_RejectsTooFew(t *testing.T) {
	spec := catalog.ProviderSpec{
		Roles: []catalog.HostRole{
			{Name: "control-plane", Count: 1},
			{Name: "worker", Count: 2},
		},
	}
	hosts := []BYOHost{
		{Role: "control-plane", Address: "1", SSHUser: "u"},
		{Role: "worker", Address: "2", SSHUser: "u"},
	}
	err := validateBYOHostsRoles(spec, hosts)
	if err == nil {
		t.Fatal("expected error for too few workers")
	}
	if !strings.Contains(err.Error(), "worker") {
		t.Errorf("error should mention the deficient role: %v", err)
	}
}

func TestValidateBYOHostsRoles_RejectsUnknownRole(t *testing.T) {
	spec := catalog.ProviderSpec{
		Roles: []catalog.HostRole{{Name: "node", Count: 1}},
	}
	hosts := []BYOHost{
		{Role: "node", Address: "1", SSHUser: "u"},
		{Role: "bastion", Address: "2", SSHUser: "u"},
	}
	err := validateBYOHostsRoles(spec, hosts)
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
	if !strings.Contains(err.Error(), "bastion") {
		t.Errorf("error should mention the unknown role: %v", err)
	}
}

func TestPickAccessHost_PrefersControlPlane(t *testing.T) {
	hosts := []BYOHost{
		{Role: "worker", Address: "10.0.0.1"},
		{Role: "control-plane", Address: "10.0.0.2"},
		{Role: "worker", Address: "10.0.0.3"},
	}
	if got := pickAccessHost(hosts); got.Address != "10.0.0.2" {
		t.Errorf("pickAccessHost = %q, want 10.0.0.2", got.Address)
	}
}

func TestPickAccessHost_FallsBackToFirst(t *testing.T) {
	hosts := []BYOHost{
		{Role: "node", Address: "10.0.0.1"},
		{Role: "node", Address: "10.0.0.2"},
	}
	if got := pickAccessHost(hosts); got.Address != "10.0.0.1" {
		t.Errorf("pickAccessHost = %q, want 10.0.0.1", got.Address)
	}
}

func TestWritePrivateKey_WritesWithRestrictedPerms(t *testing.T) {
	dir := t.TempDir()
	path, err := writePrivateKey(dir, "-----BEGIN OPENSSH PRIVATE KEY-----\nxxx\n-----END OPENSSH PRIVATE KEY-----\n")
	if err != nil {
		t.Fatalf("writePrivateKey: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("key written outside dir: %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %v, want 0600", info.Mode().Perm())
	}
}
