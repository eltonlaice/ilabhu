package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Catalog is an in-memory index of labs loaded from disk.
type Catalog struct {
	mu   sync.RWMutex
	labs map[string]*Manifest // keyed by Manifest.ID
}

// Load walks `root` looking for `lab.yaml` files and parses each one.
func Load(root string) (*Catalog, error) {
	c := &Catalog{labs: map[string]*Manifest{}}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "lab.yaml" {
			return nil
		}
		m, err := parseManifest(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if existing, ok := c.labs[m.ID]; ok {
			return fmt.Errorf("duplicate lab id %q (also defined in %s)", m.ID, existing.Dir)
		}
		c.labs[m.ID] = m
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c, nil
		}
		return nil, err
	}
	return c, nil
}

func parseManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d (expected 1)", m.SchemaVersion)
	}
	if m.ID == "" {
		return nil, fmt.Errorf("missing required field: id")
	}
	m.Dir = filepath.Dir(path)
	return &m, nil
}

// Get returns the lab with the given id, or false if not found.
func (c *Catalog) Get(id string) (*Manifest, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.labs[id]
	return m, ok
}

// List returns a snapshot of all labs.
func (c *Catalog) List() []*Manifest {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Manifest, 0, len(c.labs))
	for _, m := range c.labs {
		out = append(out, m)
	}
	return out
}
