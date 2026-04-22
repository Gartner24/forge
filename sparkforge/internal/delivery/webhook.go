package delivery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
)

type WebhookDelivery struct{}

func (w *WebhookDelivery) Deliver(ch model.Channel, msg model.Message) error {
	url := ch.Config.WebhookURL
	if url == "" {
		return fmt.Errorf("webhook_url not configured")
	}

	var payload []byte
	var err error

	switch {
	case strings.Contains(url, "hooks.slack.com"):
		payload, err = slackPayload(msg)
	case strings.Contains(url, "discord.com/api/webhooks"):
		payload, err = discordPayload(msg)
	default:
		payload, err = genericPayload(msg)
	}
	if err != nil {
		return fmt.Errorf("encoding webhook payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func genericPayload(msg model.Message) ([]byte, error) {
	return json.Marshal(map[string]any{
		"title":     msg.Title,
		"body":      msg.Body,
		"priority":  string(msg.Priority),
		"source":    msg.Source,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func slackPayload(msg model.Message) ([]byte, error) {
	emoji := priorityEmoji(msg.Priority)
	text := fmt.Sprintf("%s *[%s]* %s", emoji, strings.ToUpper(string(msg.Priority)), msg.Title)
	if msg.Body != "" {
		text += "\n" + msg.Body
	}
	if msg.Link != "" {
		text += "\n" + msg.Link
	}
	return json.Marshal(map[string]any{"text": text})
}

func discordPayload(msg model.Message) ([]byte, error) {
	emoji := priorityEmoji(msg.Priority)
	content := fmt.Sprintf("%s **[%s]** %s", emoji, strings.ToUpper(string(msg.Priority)), msg.Title)
	if msg.Body != "" {
		content += "\n" + msg.Body
	}
	if msg.Link != "" {
		content += "\n" + msg.Link
	}
	return json.Marshal(map[string]any{"content": content})
}

func priorityEmoji(p model.Priority) string {
	switch p {
	case model.PriorityCritical:
		return ":rotating_light:"
	case model.PriorityHigh:
		return ":warning:"
	case model.PriorityMedium:
		return ":bell:"
	default:
		return ":information_source:"
	}
}
