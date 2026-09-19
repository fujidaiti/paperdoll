package infra

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/fujidaiti/paperdoll/server/cfg"
	"github.com/resend/resend-go/v4"
)

type EmailDraft struct {
	To      string
	Subject string

	// The HTML body.
	Body string
}

type EmailSender interface {
	Send(d EmailDraft) error
}

func NewEmailSenderFrom(c cfg.Config) (EmailSender, error) {
	switch c.EmailTransport {
	case cfg.EmailTransportDebug:
		return &DebugSMTPClient{
			From: c.EmailFrom,
			Host: c.SMTPHost,
			Port: c.SMTPPort,
		}, nil

	case cfg.EmailTransportResend:
		return &resendClient{
			fromAddr: c.EmailFrom,
			inner:    resend.NewClient(c.ResendAPIKey),
		}, nil

	default:
		return nil, fmt.Errorf("unknown email transport method: %s", c.EmailTransport)
	}
}

// DebugSMTPClient sends emails via SMTP. Do not use in production;
// this client never cares about injection attacks.
type DebugSMTPClient struct {
	From string
	Host string
	Port string
}

func (s *DebugSMTPClient) Send(d EmailDraft) error {
	const delim = "\r\n"
	addr := s.Host + ":" + s.Port
	msg := strings.Join([]string{
		"From: " + s.From,
		"To: " + d.To,
		"Subject: " + d.Subject,
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="utf8"`,
		"", // A required empty line
		d.Body,
	}, delim) + delim
	return smtp.SendMail(addr, nil, s.From, []string{d.To}, []byte(msg))
}

type resendClient struct {
	fromAddr string
	inner    *resend.Client
}

func (c *resendClient) Send(d EmailDraft) error {
	_, err := c.inner.Emails.Send(&resend.SendEmailRequest{
		From:    c.fromAddr,
		To:      []string{d.To},
		Subject: d.Subject,
		Html:    d.Body,
	})
	return err
}
