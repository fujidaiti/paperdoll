package infra

type EmailDraft struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(d EmailDraft) error
}

type SMTPClient struct {
	HOST string
	PORT string
}

func (s *SMTPClient) Send(d EmailDraft) error {
	// TODO: send the email via SMTP
	return nil
}
