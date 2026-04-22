package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"

	sparkforgeAddr = "http://localhost:7778"
)

// Message is the notification payload.
type Message struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Priority  string `json:"priority"`
	Source    string `json:"source"`
	Link      string `json:"link,omitempty"`
	EventType string `json:"event_type,omitempty"`
}

var httpClient = &http.Client{Timeout: 3 * time.Second}

// Send delivers a notification via SparkForge. If SparkForge is not running or
// unreachable, this is a silent no-op — it returns nil without error.
func Send(token string, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", sparkforgeAddr+"/notify", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil // SparkForge not available — silent no-op
	}
	defer resp.Body.Close()
	return nil
}

// Available returns true if the SparkForge API is reachable.
func Available() bool {
	probe := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := probe.Get(sparkforgeAddr + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}
