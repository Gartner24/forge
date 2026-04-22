package registry

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
)

type Registry struct {
	path string
}

func New() (*Registry, error) {
	p, err := paths.ChannelsFile()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, fmt.Errorf("creating registry dir: %w", err)
	}
	return &Registry{path: p}, nil
}

func (r *Registry) List() ([]model.Channel, error) {
	return r.load()
}

func (r *Registry) Get(id string) (*model.Channel, error) {
	channels, err := r.load()
	if err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if ch.ID == id {
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("channel %q not found", id)
}

func (r *Registry) Add(ch model.Channel) (model.Channel, error) {
	channels, err := r.load()
	if err != nil {
		return model.Channel{}, err
	}
	for _, existing := range channels {
		if existing.Name == ch.Name {
			return model.Channel{}, fmt.Errorf("channel with name %q already exists", ch.Name)
		}
	}
	ch.ID = newID()
	ch.CreatedAt = time.Now().UTC()
	channels = append(channels, ch)
	if err := r.save(channels); err != nil {
		return model.Channel{}, err
	}
	return ch, nil
}

func (r *Registry) Update(ch model.Channel) error {
	channels, err := r.load()
	if err != nil {
		return err
	}
	for i, existing := range channels {
		if existing.ID == ch.ID {
			channels[i] = ch
			return r.save(channels)
		}
	}
	return fmt.Errorf("channel %q not found", ch.ID)
}

func (r *Registry) Delete(id string) error {
	channels, err := r.load()
	if err != nil {
		return err
	}
	next := channels[:0]
	found := false
	for _, ch := range channels {
		if ch.ID == id {
			found = true
			continue
		}
		next = append(next, ch)
	}
	if !found {
		return fmt.Errorf("channel %q not found", id)
	}
	return r.save(next)
}

func (r *Registry) SetEnabled(id string, enabled bool) error {
	channels, err := r.load()
	if err != nil {
		return err
	}
	for i, ch := range channels {
		if ch.ID == id {
			channels[i].Enabled = enabled
			return r.save(channels)
		}
	}
	return fmt.Errorf("channel %q not found", id)
}

func (r *Registry) load() ([]model.Channel, error) {
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		return []model.Channel{}, nil
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("reading channels: %w", err)
	}
	var channels []model.Channel
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("parsing channels: %w", err)
	}
	return channels, nil
}

func (r *Registry) save(channels []model.Channel) error {
	if channels == nil {
		channels = []model.Channel{}
	}
	data, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding channels: %w", err)
	}
	return os.WriteFile(r.path, data, 0644)
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return fmt.Sprintf("%x", b)
}
