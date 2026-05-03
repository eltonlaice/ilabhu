package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
	awscloud "github.com/eltonlaice/ilabhu/control-plane/internal/cloud/aws"
	"github.com/eltonlaice/ilabhu/control-plane/internal/provisioner"
)

// StartInput is the per-provider payload supplied by the user when starting a
// session. Exactly one of the provider-specific fields must be populated and
// must match Provider.
type StartInput struct {
	Provider string // aws | gcp | azure | digitalocean | byo-hosts

	AWS *AWSCredentials
	// GCP, Azure, DigitalOcean, BYOHosts to be added in follow-up PRs.
}

// AWSCredentials carries the role ARN + external id pair the control plane
// uses to assume a short-lived role in the user's account.
type AWSCredentials struct {
	RoleARN    string
	ExternalID string
}

// Manager owns the lifecycle of sessions: it talks to the catalog, provisions
// infrastructure, fetches access details, and tears everything down.
type Manager struct {
	Store    *Store
	Catalog  *catalog.Catalog
	StateDir string
	TF       *provisioner.Terraform
	Logger   *slog.Logger
}

// Start kicks off provisioning for `examID`. It returns immediately; the
// caller polls the store for status.
func (m *Manager) Start(ctx context.Context, examID string, input StartInput) (*Session, error) {
	exam, ok := m.Catalog.Get(examID)
	if !ok {
		return nil, fmt.Errorf("exam %q not found", examID)
	}
	if input.Provider == "" {
		return nil, errors.New("provider is required")
	}
	spec, ok := exam.Infrastructure.Providers[input.Provider]
	if !ok {
		return nil, fmt.Errorf("exam %q does not declare provider %q", examID, input.Provider)
	}
	switch input.Provider {
	case "aws":
		if input.AWS == nil || input.AWS.RoleARN == "" {
			return nil, errors.New("aws.role_arn is required")
		}
	default:
		return nil, fmt.Errorf("provider %q not implemented yet", input.Provider)
	}
	_ = spec // used inside provision

	sess := m.Store.Create(examID)
	_ = m.Store.Update(sess.ID, func(s *Session) { s.Provider = input.Provider })
	go m.provision(context.Background(), sess, exam, spec, input)
	return sess, nil
}

func (m *Manager) provision(ctx context.Context, sess *Session, exam *catalog.Manifest, spec catalog.ProviderSpec, input StartInput) {
	log := m.Logger.With("session_id", sess.ID, "exam_id", exam.ID, "provider", input.Provider)

	workdir := filepath.Join(m.StateDir, "sessions", sess.ID)
	if err := os.MkdirAll(workdir, 0o700); err != nil {
		m.fail(sess, fmt.Errorf("mkdir workdir: %w", err))
		return
	}
	moduleDst := filepath.Join(workdir, "module")
	moduleSrc := filepath.Join(exam.Dir, spec.Module)
	if err := provisioner.PrepareWorkdir(moduleSrc, moduleDst); err != nil {
		m.fail(sess, fmt.Errorf("copy module: %w", err))
		return
	}

	keyPath := filepath.Join(workdir, "ssh_key")
	pubKey, err := generateSSHKey(keyPath)
	if err != nil {
		m.fail(sess, fmt.Errorf("generate ssh key: %w", err))
		return
	}

	awsCreds, err := awscloud.AssumeRole(ctx, input.AWS.RoleARN, input.AWS.ExternalID, "ilabhu-"+sess.ID, time.Hour)
	if err != nil {
		m.fail(sess, fmt.Errorf("assume role: %w", err))
		return
	}

	vars := map[string]any{
		"session_id":     sess.ID,
		"ssh_public_key": pubKey,
	}
	for k, v := range spec.Inputs {
		vars[k] = v
	}

	log.Info("provisioning")
	_ = m.Store.Update(sess.ID, func(s *Session) {
		s.Workdir = workdir
		s.SSHPrivateKeyPath = keyPath
	})

	if err := m.TF.Apply(ctx, moduleDst, vars, awsCreds.AsEnv()); err != nil {
		m.fail(sess, fmt.Errorf("terraform apply: %w", err))
		return
	}
	outs, err := m.TF.Outputs(ctx, moduleDst, awsCreds.AsEnv())
	if err != nil {
		m.fail(sess, fmt.Errorf("read outputs: %w", err))
		return
	}
	plain := map[string]any{}
	for k, v := range outs {
		plain[k] = v.Value
	}

	if exam.Access.Kind == "kubeconfig" {
		ip, _ := plain["public_ip"].(string)
		user, _ := plain["ssh_user"].(string)
		remotePath, _ := plain["kubeconfig_path_on_host"].(string)
		if ip == "" || user == "" || remotePath == "" {
			m.fail(sess, errors.New("exam outputs missing public_ip/ssh_user/kubeconfig_path_on_host"))
			return
		}
		kubeconfig, err := fetchKubeconfig(ctx, keyPath, user, ip, remotePath)
		if err != nil {
			m.fail(sess, fmt.Errorf("fetch kubeconfig: %w", err))
			return
		}
		_ = m.Store.Update(sess.ID, func(s *Session) {
			s.Kubeconfig = kubeconfig
		})
	}

	_ = m.Store.Update(sess.ID, func(s *Session) {
		s.Status = StatusReady
		s.Outputs = plain
	})
	log.Info("ready")
}

// Destroy tears down a session synchronously.
func (m *Manager) Destroy(ctx context.Context, sessID string, input StartInput) error {
	sess, err := m.Store.Get(sessID)
	if err != nil {
		return err
	}
	if sess.Workdir == "" {
		return errors.New("session has no workdir; nothing to destroy")
	}
	if input.Provider != sess.Provider {
		return fmt.Errorf("provider mismatch: session was started with %q, got %q", sess.Provider, input.Provider)
	}
	switch input.Provider {
	case "aws":
		if input.AWS == nil || input.AWS.RoleARN == "" {
			return errors.New("aws.role_arn is required")
		}
		awsCreds, err := awscloud.AssumeRole(ctx, input.AWS.RoleARN, input.AWS.ExternalID, "ilabhu-destroy-"+sess.ID, time.Hour)
		if err != nil {
			return fmt.Errorf("assume role: %w", err)
		}
		_ = m.Store.Update(sessID, func(s *Session) { s.Status = StatusDestroying })
		moduleDst := filepath.Join(sess.Workdir, "module")
		if err := m.TF.Destroy(ctx, moduleDst, awsCreds.AsEnv()); err != nil {
			_ = m.Store.Update(sessID, func(s *Session) {
				s.Status = StatusFailed
				s.Error = err.Error()
			})
			return err
		}
	default:
		return fmt.Errorf("provider %q not implemented yet", input.Provider)
	}
	_ = m.Store.Update(sessID, func(s *Session) { s.Status = StatusDestroyed })
	return nil
}

// Get returns the session record for `id`, or session.ErrNotFound. It is a
// thin convenience over Manager.Store so callers (notably the API layer) do
// not need to depend on the in-memory store implementation directly.
func (m *Manager) Get(id string) (*Session, error) {
	return m.Store.Get(id)
}

func (m *Manager) fail(sess *Session, err error) {
	m.Logger.Error("session failed", "session_id", sess.ID, "err", err)
	_ = m.Store.Update(sess.ID, func(s *Session) {
		s.Status = StatusFailed
		s.Error = err.Error()
	})
}

// generateSSHKey shells out to ssh-keygen to create an ed25519 keypair.
// Returns the public key contents.
func generateSSHKey(path string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", path, "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	pub, err := os.ReadFile(path + ".pub")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(pub)), nil
}

// fetchKubeconfig polls the lab VM via SSH until /home/ubuntu/kubeconfig
// exists, then SCPs it down. The VM's user_data installs k3s and writes the
// file once the API server is up.
func fetchKubeconfig(ctx context.Context, keyPath, user, host, remotePath string) ([]byte, error) {
	deadline := time.Now().Add(5 * time.Minute)
	sshOpts := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
	}
	target := user + "@" + host

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		args := append([]string{}, sshOpts...)
		args = append(args, target, "test", "-f", remotePath)
		if err := exec.CommandContext(ctx, "ssh", args...).Run(); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	tmp, err := os.CreateTemp("", "ilabhu-kubeconfig-*")
	if err != nil {
		return nil, err
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmp.Name()) }()

	scpArgs := append([]string{}, sshOpts...)
	scpArgs = append(scpArgs, target+":"+remotePath, tmp.Name())
	if out, err := exec.CommandContext(ctx, "scp", scpArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("scp: %w: %s", err, out)
	}
	return os.ReadFile(tmp.Name())
}
