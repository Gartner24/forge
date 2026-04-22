package router

import (
	"fmt"
	"time"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/sparkforge/internal/dedup"
	"github.com/gartner24/forge/sparkforge/internal/deliverylog"
	"github.com/gartner24/forge/sparkforge/internal/delivery"
	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
	"github.com/gartner24/forge/sparkforge/internal/registry"
)

type Router struct {
	reg    *registry.Registry
	log    *deliverylog.Log
	dedup  *dedup.Store
	secrets *secrets.Store
	gotify  *delivery.GotifyDelivery
	email   *delivery.EmailDelivery
	webhook *delivery.WebhookDelivery
}

func New() (*Router, error) {
	reg, err := registry.New()
	if err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	dl, err := deliverylog.New()
	if err != nil {
		return nil, fmt.Errorf("delivery log: %w", err)
	}
	dd, err := dedup.New()
	if err != nil {
		return nil, fmt.Errorf("dedup store: %w", err)
	}
	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	sec, err := secrets.New(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("secrets: %w", err)
	}
	return &Router{
		reg:     reg,
		log:     dl,
		dedup:   dd,
		secrets: sec,
		gotify:  &delivery.GotifyDelivery{},
		email:   &delivery.EmailDelivery{},
		webhook: &delivery.WebhookDelivery{},
	}, nil
}

// Send routes a message to all matching channels and returns the names of
// channels that received it successfully.
func (r *Router) Send(msg model.Message) ([]string, error) {
	if !msg.Priority.Valid() {
		return nil, fmt.Errorf("invalid priority %q", msg.Priority)
	}

	// Deduplication: if event_type is set and an active alert exists, drop silently.
	if msg.EventType != "" && r.dedup.IsDuplicate(msg.Source, msg.EventType) {
		return nil, nil
	}

	// Track the alert if it has an event_type.
	if msg.EventType != "" {
		if _, err := r.dedup.Add(msg); err != nil {
			// Non-fatal: log and continue.
			fmt.Printf("sparkforge: failed to record alert: %v\n", err)
		}
	}

	channels, err := r.reg.List()
	if err != nil {
		return nil, fmt.Errorf("loading channels: %w", err)
	}

	var delivered []string
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		if msg.Priority.Level() < ch.PriorityMin.Level() {
			continue
		}
		deliveryErr := r.deliverToChannel(ch, msg)
		status := "ok"
		errMsg := ""
		if deliveryErr != nil {
			status = "error"
			errMsg = deliveryErr.Error()
		} else {
			delivered = append(delivered, ch.Name)
		}
		_ = r.log.Append(model.DeliveryRecord{
			Timestamp:   time.Now().UTC(),
			ChannelID:   ch.ID,
			ChannelName: ch.Name,
			Title:       msg.Title,
			Priority:    msg.Priority,
			Source:      msg.Source,
			Status:      status,
			Error:       errMsg,
		})
	}
	return delivered, nil
}

// SendToChannel routes a message to a single channel by ID.
func (r *Router) SendToChannel(channelID string, msg model.Message) ([]string, error) {
	if !msg.Priority.Valid() {
		return nil, fmt.Errorf("invalid priority %q", msg.Priority)
	}
	ch, err := r.reg.Get(channelID)
	if err != nil {
		return nil, err
	}
	if !ch.Enabled {
		return nil, fmt.Errorf("channel %q is disabled", ch.Name)
	}
	deliveryErr := r.deliverToChannel(*ch, msg)
	status := "ok"
	errMsg := ""
	if deliveryErr != nil {
		status = "error"
		errMsg = deliveryErr.Error()
	}
	_ = r.log.Append(model.DeliveryRecord{
		Timestamp:   time.Now().UTC(),
		ChannelID:   ch.ID,
		ChannelName: ch.Name,
		Title:       msg.Title,
		Priority:    msg.Priority,
		Source:      msg.Source,
		Status:      status,
		Error:       errMsg,
	})
	if deliveryErr != nil {
		return nil, deliveryErr
	}
	return []string{ch.Name}, nil
}

func (r *Router) deliverToChannel(ch model.Channel, msg model.Message) error {
	switch ch.Type {
	case model.ChannelTypeGotify:
		token, _ := r.secrets.Get(fmt.Sprintf("sparkforge.channels.%s.gotify_token", ch.ID))
		return r.gotify.Deliver(ch, msg, token)
	case model.ChannelTypeEmail:
		pass, _ := r.secrets.Get(fmt.Sprintf("sparkforge.channels.%s.smtp_password", ch.ID))
		return r.email.Deliver(ch, msg, pass)
	case model.ChannelTypeWebhook:
		return r.webhook.Deliver(ch, msg)
	default:
		return fmt.Errorf("unknown channel type %q", ch.Type)
	}
}
