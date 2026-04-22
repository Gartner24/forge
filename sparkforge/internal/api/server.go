package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
	"github.com/gartner24/forge/sparkforge/internal/router"
)

const Addr = "localhost:7778"

type Server struct {
	router  *router.Router
	secrets *secrets.Store
}

type notifyRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Priority  string `json:"priority"`
	Source    string `json:"source"`
	Link      string `json:"link"`
	EventType string `json:"event_type"`
}

type notifyResponse struct {
	Status   string   `json:"status"`
	Channels []string `json:"channels"`
}

func New(r *router.Router) (*Server, error) {
	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	sec, err := secrets.New(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("secrets: %w", err)
	}
	return &Server{router: r, secrets: sec}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify", s.handleNotify)
	mux.HandleFunc("GET /health", s.handleHealth)
	fmt.Printf("sparkforge: listening on %s\n", Addr)
	return http.ListenAndServe(Addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.authenticate(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "title is required"})
		return
	}
	if req.Source == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "source is required"})
		return
	}

	priority, err := model.ParsePriority(req.Priority)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	msg := model.Message{
		Title:     req.Title,
		Body:      req.Body,
		Priority:  priority,
		Source:    req.Source,
		Link:      req.Link,
		EventType: req.EventType,
	}

	channels, err := s.router.Send(msg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if channels == nil {
		channels = []string{}
	}

	status := "delivered"
	if len(channels) == 0 {
		status = "no_channels"
	}

	json.NewEncoder(w).Encode(notifyResponse{
		Status:   status,
		Channels: channels,
	})
}

func (s *Server) authenticate(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return false
	}

	keys, err := s.secrets.List()
	if err != nil {
		return false
	}
	for _, key := range keys {
		if !strings.HasPrefix(key, "sparkforge.api_tokens.") {
			continue
		}
		val, err := s.secrets.Get(key)
		if err != nil {
			continue
		}
		if val == token {
			return true
		}
	}
	return false
}
