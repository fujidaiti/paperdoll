package infra

type EmailDraft struct {
	To      string
	Subject string
	Body    string
}

type EmailSender = func(d EmailDraft) error

func SendEmail(d EmailDraft) error {
	// TODO: Send an email using Resend
	return nil
}
