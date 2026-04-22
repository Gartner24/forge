package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gartner24/forge/smeltforge/internal/deploy"
	"github.com/gartner24/forge/smeltforge/internal/registry"
)

// DeployFunc is called to trigger a deploy. The caller provides project ID and trigger label.
type DeployFunc func(ctx context.Context, projectID, trigger string) (*deploy.Result, error)

// SecretLookup returns the stored secret for a given key.
type SecretLookup func(key string) (string, error)

// Server handles incoming webhook and CI token deploy requests.
type Server struct {
	mux          *http.ServeMux
	reg          *registry.Registry
	deployFn     DeployFunc
	secretLookup SecretLookup
	queue        chan deployRequest
}

type deployRequest struct {
	projectID string
	trigger   string
}

const defaultQueueDepth = 5

func New(reg *registry.Registry, deployFn DeployFunc, secretLookup SecretLookup) *Server {
	s := &Server{
		mux:          http.NewServeMux(),
		reg:          reg,
		deployFn:     deployFn,
		secretLookup: secretLookup,
		queue:        make(chan deployRequest, defaultQueueDepth),
	}
	s.mux.HandleFunc("/_smeltforge/webhook/", s.handleWebhook)
	s.mux.HandleFunc("/_smeltforge/deploy", s.handleCIDeploy)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleWebhook handles: POST /_smeltforge/webhook/<project>/<secret>
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse /_smeltforge/webhook/<project>/<secret>
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/_smeltforge/webhook/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "invalid webhook path", http.StatusBadRequest)
		return
	}
	projectID, providedSecret := parts[0], parts[1]

	storedSecret, err := s.secretLookup("smeltforge." + projectID + "._webhook")
	if err != nil || subtle.ConstantTimeCompare([]byte(storedSecret), []byte(providedSecret)) != 1 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := s.enqueue(projectID, "webhook"); err != nil {
		http.Error(w, "deploy queue full", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "project": projectID})
}

// handleCIDeploy handles: POST /_smeltforge/deploy with Authorization: Bearer <token>
func (s *Server) handleCIDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	providedToken := strings.TrimPrefix(authHeader, "Bearer ")

	var body struct {
		Project string `json:"project"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Project == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate token against any stored CI token for this project.
	p, err := s.reg.Get(body.Project)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	valid := false
	for _, t := range p.CITokens {
		stored, err := s.secretLookup("smeltforge." + body.Project + "._citoken_" + t.ID)
		if err == nil && subtle.ConstantTimeCompare([]byte(stored), []byte(providedToken)) == 1 {
			valid = true
			break
		}
	}
	if !valid {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := s.enqueue(body.Project, "ci-token"); err != nil {
		http.Error(w, "deploy queue full", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "project": body.Project})
}

func (s *Server) enqueue(projectID, trigger string) error {
	select {
	case s.queue <- deployRequest{projectID: projectID, trigger: trigger}:
		return nil
	default:
		return fmt.Errorf("queue full")
	}
}

// RunWorker processes queued deploys sequentially. Blocks until ctx is cancelled.
func (s *Server) RunWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.queue:
			deployCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			_, _ = s.deployFn(deployCtx, req.projectID, req.trigger)
			cancel()
		}
	}
}

// RunPoller polls git projects at their configured interval. Blocks until ctx is cancelled.
func (s *Server) RunPoller(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	lastPolled := map[string]time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for _, p := range s.reg.All() {
				if !p.PollingOn || p.Source.Type != "git" {
					continue
				}
				interval := p.Trigger.Interval
				if interval <= 0 {
					interval = 60 // default 60s
				}
				last := lastPolled[p.ID]
				if now.Sub(last) < time.Duration(interval)*time.Second {
					continue
				}
				lastPolled[p.ID] = now
				_ = s.enqueue(p.ID, "polling")
			}
		}
	}
}
