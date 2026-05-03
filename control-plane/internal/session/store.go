package session

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status is the current state of a lab session.
type Status string

const (
	StatusProvisioning Status = "provisioning"
	StatusReady        Status = "ready"
	StatusFailed       Status = "failed"
	StatusDestroying   Status = "destroying"
	StatusDestroyed    Status = "destroyed"
)

// Session is the in-memory record for a single lab session.
type Session struct {
	ID        string    `json:"id"`
	LabID     string    `json:"lab_id"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Workdir is the temp directory holding the per-session terraform state
	// and module copy.
	Workdir string `json:"-"`

	// Outputs are the parsed terraform outputs once the session is ready.
	Outputs map[string]any `json:"outputs,omitempty"`

	// Kubeconfig is the contents of the cluster kubeconfig fetched after
	// provisioning, if the lab's access kind is "kubeconfig".
	Kubeconfig []byte `json:"-"`

	// Error is set when Status is StatusFailed.
	Error string `json:"error,omitempty"`

	// SSHPrivateKeyPath is the path to the per-session private key the
	// control plane uses to reach the lab VM.
	SSHPrivateKeyPath string `json:"-"`
}

// Store is an in-memory, goroutine-safe registry of sessions.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewStore() *Store {
	return &Store{sessions: map[string]*Session{}}
}

var ErrNotFound = errors.New("session not found")

// Create allocates a new session in the StatusProvisioning state.
func (s *Store) Create(labID string) *Session {
	now := time.Now().UTC()
	sess := &Session{
		ID:        uuid.NewString(),
		LabID:     labID,
		Status:    StatusProvisioning,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()
	return sess
}

func (s *Store) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return sess, nil
}

// Update applies fn under the store lock so callers don't have to manage it.
// fn must not block on external IO.
func (s *Store) Update(id string, fn func(*Session)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return ErrNotFound
	}
	fn(sess)
	sess.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		out = append(out, sess)
	}
	return out
}
