package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Source struct {
	Type   string `json:"type"`
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Image  string `json:"image,omitempty"`
	Path   string `json:"path,omitempty"`
}

type HealthCheck struct {
	Path     string `json:"path"`
	Timeout  int    `json:"timeout"`
	Interval int    `json:"interval"`
}

type Trigger struct {
	Type     string `json:"type"`
	Interval int    `json:"interval,omitempty"`
	Branch   string `json:"branch,omitempty"`
}

// ContainerState holds runtime state tracked alongside project config.
type ContainerState struct {
	ID            string `json:"id,omitempty"`
	Image         string `json:"image,omitempty"`
	Color         string `json:"color,omitempty"` // "blue" or "green" for blue-green
	PreviousID    string `json:"previous_id,omitempty"`
	PreviousImage string `json:"previous_image,omitempty"`
}

type CIToken struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type Project struct {
	ID          string         `json:"id"`
	Source      Source         `json:"source"`
	Domain      string         `json:"domain"`
	Port        int            `json:"port"`
	Strategy    string         `json:"strategy"` // "stop-start" or "blue-green"
	HealthCheck HealthCheck    `json:"health_check,omitempty"`
	Trigger     Trigger        `json:"trigger"`
	Watch       bool           `json:"watch"`
	State       ContainerState `json:"state,omitempty"`
	CITokens    []CIToken      `json:"ci_tokens,omitempty"`
	PollingOn   bool           `json:"polling_on,omitempty"`
}

type Registry struct {
	mu       sync.Mutex
	path     string
	Projects []*Project `json:"projects"`
}

func Load(path string) (*Registry, error) {
	r := &Registry{path: path, Projects: []*Project{}}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return r, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	if err := json.Unmarshal(data, &r.Projects); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	return r, nil
}

func (r *Registry) save() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("creating registry dir: %w", err)
	}
	data, err := json.MarshalIndent(r.Projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

func (r *Registry) Add(p *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.Projects {
		if existing.ID == p.ID {
			return fmt.Errorf("project %q already exists", p.ID)
		}
	}
	r.Projects = append(r.Projects, p)
	return r.save()
}

func (r *Registry) Get(id string) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.Projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", id)
}

func (r *Registry) Update(p *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.Projects {
		if existing.ID == p.ID {
			r.Projects[i] = p
			return r.save()
		}
	}
	return fmt.Errorf("project %q not found", p.ID)
}

func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.Projects {
		if p.ID == id {
			r.Projects = append(r.Projects[:i], r.Projects[i+1:]...)
			return r.save()
		}
	}
	return fmt.Errorf("project %q not found", id)
}

func (r *Registry) All() []*Project {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Project, len(r.Projects))
	copy(out, r.Projects)
	return out
}
