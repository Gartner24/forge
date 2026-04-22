package hook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/gartner24/forge/penforge/internal/scanner"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/gartner24/forge/shared/registry"
)

const hookAddr = "localhost:7774"

type postDeployPayload struct {
	TargetID string `json:"target_id"`
}

// Server is the post-deploy HTTP hook server called by SmeltForge after each deploy.
type Server struct {
	targetsPath string
	findings    *store.FindingStore
	scans       *store.ScanStore
	sparkAddr   string
	auditPath   string
}

func New(targetsPath string, findings *store.FindingStore, scans *store.ScanStore, sparkAddr, auditPath string) *Server {
	return &Server{
		targetsPath: targetsPath,
		findings:    findings,
		scans:       scans,
		sparkAddr:   sparkAddr,
		auditPath:   auditPath,
	}
}

// ListenAndServe starts the hook HTTP server at localhost:7774.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hook/post-deploy", s.handlePostDeploy)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	log.Printf("penforge: hook server listening on %s", hookAddr)
	return http.ListenAndServe(hookAddr, mux)
}

func (s *Server) handlePostDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload postDeployPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if payload.TargetID == "" {
		http.Error(w, "target_id required", http.StatusBadRequest)
		return
	}

	targets, err := registry.ReadScanTargets(s.targetsPath)
	if err != nil {
		http.Error(w, "reading targets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var target *registry.ScanTarget
	for _, t := range targets {
		if t.ID == payload.TargetID {
			cp := t
			target = &cp
			break
		}
	}
	if target == nil {
		http.Error(w, fmt.Sprintf("target %q not registered", payload.TargetID), http.StatusNotFound)
		return
	}

	engines := engine.ForTarget(*target)
	scanID := newScanID()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"scan_id":           scanID,
		"target_id":         target.ID,
		"engines":           engineNames(engines),
		"estimated_seconds": len(engines) * 120,
	})

	auditPath := s.auditPath
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if _, err := scanner.RunScan(ctx, scanID, *target, engines, s.findings, s.scans, s.sparkAddr, auditPath); err != nil {
			log.Printf("penforge: post-deploy scan failed for %s: %v", target.ID, err)
			scanner.MarkFailed(s.scans, scanID, err)
		}
	}()
}

func engineNames(engines []engine.Engine) []string {
	names := make([]string, len(engines))
	for i, e := range engines {
		names[i] = e.Name()
	}
	return names
}

func newScanID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("scan-%s-%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b))
}
