package infra

import (
	"net/smtp"
	"strings"

	"github.com/resend/resend-go/v4"
)

type EmailDraft struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(d EmailDraft) error
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

// ResendClient is the client for Resend (https://resend.com).
type ResendClient struct {
	From   string
	APIKey string
}

func (s *ResendClient) Send(d EmailDraft) error {
	client := resend.NewClient(s.APIKey)
	_, err := client.Emails.Send(&resend.SendEmailRequest{
		From:    s.From,
		To:      []string{d.To},
		Subject: d.Subject,
		Html:    d.Body,
	})
	return err
}
