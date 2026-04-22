package delivery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gartner24/forge/sparkforge/internal/model"
)

type GotifyDelivery struct{}

func (g *GotifyDelivery) Deliver(ch model.Channel, msg model.Message, token string) error {
	url := ch.Config.GotifyURL
	if url == "" {
		url = "http://localhost:7779"
	}

	priority := gotifyPriority(msg.Priority)

	body := msg.Body
	if msg.Link != "" {
		if body != "" {
			body += "\n\n" + msg.Link
		} else {
			body = msg.Link
		}
	}

	payload := map[string]any{
		"title":    msg.Title,
		"message":  body,
		"priority": priority,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding gotify payload: %w", err)
	}

	req, err := http.NewRequest("POST", url+"/message", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building gotify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending to gotify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("gotify returned %d", resp.StatusCode)
	}
	return nil
}

func gotifyPriority(p model.Priority) int {
	switch p {
	case model.PriorityLow:
		return 1
	case model.PriorityMedium:
		return 5
	case model.PriorityHigh:
		return 7
	case model.PriorityCritical:
		return 10
	default:
		return 5
	}
}
