package session

import (
	"errors"
	"sync"
	"testing"
)

func TestStore_CreateAssignsIDAndState(t *testing.T) {
	s := NewStore()
	sess := s.Create("cka/example")

	if sess.ID == "" {
		t.Error("ID is empty")
	}
	if sess.LabID != "cka/example" {
		t.Errorf("LabID = %q, want cka/example", sess.LabID)
	}
	if sess.Status != StatusProvisioning {
		t.Errorf("Status = %q, want %q", sess.Status, StatusProvisioning)
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	if !sess.UpdatedAt.Equal(sess.CreatedAt) {
		t.Errorf("UpdatedAt %v should equal CreatedAt %v on create", sess.UpdatedAt, sess.CreatedAt)
	}
}

func TestStore_GetReturnsErrNotFoundForUnknownID(t *testing.T) {
	s := NewStore()
	_, err := s.Get("does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStore_GetReturnsCreatedSession(t *testing.T) {
	s := NewStore()
	sess := s.Create("cka/example")

	got, err := s.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("Get returned different ID: %q vs %q", got.ID, sess.ID)
	}
}

func TestStore_UpdateAppliesMutationAndBumpsTimestamp(t *testing.T) {
	s := NewStore()
	sess := s.Create("cka/example")
	originalUpdatedAt := sess.UpdatedAt

	err := s.Update(sess.ID, func(s *Session) {
		s.Status = StatusReady
		s.Outputs = map[string]any{"public_ip": "1.2.3.4"}
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := s.Get(sess.ID)
	if got.Status != StatusReady {
		t.Errorf("Status = %q, want %q", got.Status, StatusReady)
	}
	if got.Outputs["public_ip"] != "1.2.3.4" {
		t.Errorf("Outputs[public_ip] = %v, want 1.2.3.4", got.Outputs["public_ip"])
	}
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("UpdatedAt %v should be after %v", got.UpdatedAt, originalUpdatedAt)
	}
}

func TestStore_UpdateUnknownIDReturnsErrNotFound(t *testing.T) {
	s := NewStore()
	err := s.Update("missing", func(*Session) {})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStore_ListReturnsAllSessions(t *testing.T) {
	s := NewStore()
	a := s.Create("cka/a")
	b := s.Create("cka/b")

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}
	ids := map[string]bool{}
	for _, sess := range list {
		ids[sess.ID] = true
	}
	if !ids[a.ID] || !ids[b.ID] {
		t.Errorf("List did not contain both created sessions: %v", list)
	}
}

func TestStore_ConcurrentAccessIsSafe(t *testing.T) {
	s := NewStore()
	sess := s.Create("cka/example")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = s.Update(sess.ID, func(s *Session) {
				s.Error = "touched"
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Get(sess.ID)
		}()
	}
	wg.Wait()

	got, err := s.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get after concurrent updates: %v", err)
	}
	if got.Error != "touched" {
		t.Errorf("Error = %q, want touched", got.Error)
	}
}
