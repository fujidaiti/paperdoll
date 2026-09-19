package infra

import (
	"net/smtp"
	"strings"
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
