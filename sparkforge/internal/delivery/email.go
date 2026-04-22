package delivery

import (
	"fmt"
	"net/smtp"

	"github.com/gartner24/forge/sparkforge/internal/model"
)

type EmailDelivery struct{}

func (e *EmailDelivery) Deliver(ch model.Channel, msg model.Message, smtpPassword string) error {
	cfg := ch.Config
	if cfg.SMTPHost == "" {
		return fmt.Errorf("smtp_host not configured")
	}
	if cfg.SMTPTo == "" {
		return fmt.Errorf("smtp_to not configured")
	}

	port := cfg.SMTPPort
	if port == 0 {
		port = 587
	}

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, port)

	var auth smtp.Auth
	if cfg.SMTPUser != "" && smtpPassword != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, smtpPassword, cfg.SMTPHost)
	}

	subject := fmt.Sprintf("[%s] %s", msg.Priority, msg.Title)
	body := msg.Body
	if msg.Link != "" {
		if body != "" {
			body += "\n\n" + msg.Link
		} else {
			body = msg.Link
		}
	}

	from := cfg.SMTPUser
	if from == "" {
		from = "sparkforge@localhost"
	}

	rawMsg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		from, cfg.SMTPTo, subject, body,
	))

	if err := smtp.SendMail(addr, auth, from, []string{cfg.SMTPTo}, rawMsg); err != nil {
		return fmt.Errorf("smtp sendmail: %w", err)
	}
	return nil
}
