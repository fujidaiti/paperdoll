package infra

type Draft struct {
	To      string
	Subject string
	Body    string
}

type EmailSender = func(d Draft) error

func SendEmail(d Draft) error {
	// TODO: Send an email using Resend
	return nil
}
