package model

import (
	"fmt"
	"time"
)

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

func (p Priority) Level() int {
	switch p {
	case PriorityLow:
		return 1
	case PriorityMedium:
		return 2
	case PriorityHigh:
		return 3
	case PriorityCritical:
		return 4
	default:
		return 0
	}
}

func (p Priority) Valid() bool { return p.Level() > 0 }

func ParsePriority(s string) (Priority, error) {
	p := Priority(s)
	if !p.Valid() {
		return "", fmt.Errorf("invalid priority %q: must be low, medium, high, or critical", s)
	}
	return p, nil
}

type ChannelType string

const (
	ChannelTypeGotify  ChannelType = "gotify"
	ChannelTypeEmail   ChannelType = "email"
	ChannelTypeWebhook ChannelType = "webhook"
)

func (ct ChannelType) Valid() bool {
	return ct == ChannelTypeGotify || ct == ChannelTypeEmail || ct == ChannelTypeWebhook
}

type ChannelConfig struct {
	// gotify
	GotifyURL string `json:"gotify_url,omitempty"`

	// email
	SMTPHost string `json:"smtp_host,omitempty"`
	SMTPPort int    `json:"smtp_port,omitempty"`
	SMTPUser string `json:"smtp_user,omitempty"`
	SMTPTo   string `json:"smtp_to,omitempty"`

	// webhook
	WebhookURL string `json:"webhook_url,omitempty"`
}

type Channel struct {
	ID          string        `json:"id"`
	Type        ChannelType   `json:"type"`
	Name        string        `json:"name"`
	Enabled     bool          `json:"enabled"`
	PriorityMin Priority      `json:"priority_min"`
	Config      ChannelConfig `json:"config"`
	CreatedAt   time.Time     `json:"created_at"`
}

type Message struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Priority  Priority `json:"priority"`
	Source    string   `json:"source"`
	Link      string   `json:"link,omitempty"`
	EventType string   `json:"event_type,omitempty"`
}

type DeliveryRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	Title       string    `json:"title"`
	Priority    Priority  `json:"priority"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
}

type Alert struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	EventType string    `json:"event_type"`
	Title     string    `json:"title"`
	Priority  Priority  `json:"priority"`
	FiredAt   time.Time `json:"fired_at"`
}
