package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
)

// validateBYOHostsRoles checks that the user supplied at least the declared
// number of hosts for every role the exam's byo-hosts spec requires. Extra
// hosts are allowed (use cases: idle workers; the exam author can tighten
// later if needed). Hosts without a matching declared role are rejected.
func validateBYOHostsRoles(spec catalog.ProviderSpec, hosts []BYOHost) error {
	declared := map[string]int{}
	for _, r := range spec.Roles {
		declared[r.Name] = r.Count
	}
	got := map[string]int{}
	for _, h := range hosts {
		if _, ok := declared[h.Role]; !ok {
			return fmt.Errorf("host %s has role %q but the exam declares no such role", h.Address, h.Role)
		}
		got[h.Role]++
	}
	for name, want := range declared {
		if got[name] < want {
			return fmt.Errorf("role %q needs %d host(s); got %d", name, want, got[name])
		}
	}
	return nil
}

// pickAccessHost returns the host the control plane should fetch the
// kubeconfig (or other access artefact) from. Convention: if a role named
// "control-plane" exists, use the first host with that role; otherwise use
// the first host in the list. Exam authors who want a specific access host
// should declare the control-plane role.
func pickAccessHost(hosts []BYOHost) BYOHost {
	for _, h := range hosts {
		if h.Role == "control-plane" {
			return h
		}
	}
	return hosts[0]
}

// writePrivateKey writes the user-supplied SSH private key into workdir
// (mode 0600) and returns the file path. The caller is responsible for
// removing it after teardown.
func writePrivateKey(workdir, keyContent string) (string, error) {
	path := filepath.Join(workdir, "byo_ssh_key")
	if err := os.WriteFile(path, []byte(keyContent), 0o600); err != nil {
		return "", fmt.Errorf("write ssh key: %w", err)
	}
	return path, nil
}

// runScriptOnHost SCPs scriptPath onto host, makes it executable, runs it,
// and returns any error. Stdout/stderr are tee'd through the manager's
// logger.
func runScriptOnHost(ctx context.Context, log *slog.Logger, keyPath, scriptPath string, host BYOHost) error {
	sshOpts := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
	}
	target := host.SSHUser + "@" + host.Address
	remote := "/tmp/ilabhu-script.sh"

	scpArgs := append([]string{}, sshOpts...)
	scpArgs = append(scpArgs, scriptPath, target+":"+remote)
	if out, err := exec.CommandContext(ctx, "scp", scpArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("scp script to %s: %w: %s", host.Address, err, out)
	}

	sshArgs := append([]string{}, sshOpts...)
	sshArgs = append(sshArgs, target, "chmod", "+x", remote, "&&", "sudo", "-E", remote)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run script on %s: %w: %s", host.Address, err, out)
	}
	log.Info("byo-hosts script ok", "host", host.Address, "role", host.Role)
	return nil
}

// runScriptOnAllHosts runs scriptPath on every host. Errors short-circuit
// (a failed setup leaves the cluster half-built; the user can re-run after
// fixing the host).
func runScriptOnAllHosts(ctx context.Context, log *slog.Logger, keyPath, scriptPath string, hosts []BYOHost) error {
	for _, h := range hosts {
		if err := runScriptOnHost(ctx, log, keyPath, scriptPath, h); err != nil {
			return err
		}
	}
	return nil
}

// teardownOnAllHosts is best-effort: every host is attempted regardless of
// earlier failures, and the joined error captures everything that went
// wrong.
func teardownOnAllHosts(ctx context.Context, log *slog.Logger, keyPath, scriptPath string, hosts []BYOHost) error {
	var errs []error
	for _, h := range hosts {
		if err := runScriptOnHost(ctx, log, keyPath, scriptPath, h); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
