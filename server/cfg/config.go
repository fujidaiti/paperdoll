package cfg

import (
	"errors"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds the app's configuration. All environment variable names must start with
// the prefix "APP_" followed by the name given in the env tags, e.g., APP_EMAIL_HOST.
type Config struct {
	// The method used to send emails. Possible values are:
	//
	//  - [EmailTransportDebug], which uses SMTP to communicate with a development
	//    mail server such as Mailpit. Do not use this method in production as it
	//    does not protect emails against injection attacks.
	//
	//  - [EmailTransportResend], which uses Resend (https://resend.com).
	EmailTransport EmailTransport `env:"EMAIL_TRANSPORT,required"`

	// The From address used when sending an email.
	EmailFrom string `env:"EMAIL_FROM,required"`

	// The SMTP server host to which the app sends emails.
	// Required when [Config.EmailTransport] is [EmailTransportDebug].
	SMTPHost string `env:"SMTP_HOST"`

	// The SMTP port of the mail server specified to [Config.SMTPHost].
	// Required when [Config.EmailTransport] is [EmailTransportDebug].
	SMTPPort string `env:"SMTP_PORT"`

	// The API key for Resend.
	// Required when [Config.EmailTransport] is [EmailTransportResend].
	ResendAPIKey string `env:"RESEND_API_KEY"`
}

func Read() (Config, error) {
	cfg := Config{}
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: "APP_"}); err != nil {
		return Config{}, err
	}
	if err := validateEmailTransport(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type EmailTransport string

const (
	EmailTransportDebug  EmailTransport = "dbg"
	EmailTransportResend EmailTransport = "resend"
)

func validateEmailTransport(cfg *Config) error {
	switch cfg.EmailTransport {
	case EmailTransportDebug:
		if len(cfg.SMTPHost) == 0 {
			return fmt.Errorf(
				"SMTP host is required when email transport method is %q, got %q",
				EmailTransportDebug, cfg.SMTPHost,
			)
		}
		if len(cfg.SMTPPort) == 0 {
			return fmt.Errorf(
				"SMTP port is required when email transport method is %q, got %q",
				EmailTransportDebug, cfg.SMTPPort,
			)
		}

	case EmailTransportResend:
		if len(cfg.ResendAPIKey) == 0 {
			return fmt.Errorf("API key for Resend is required when email transport method is %q", EmailTransportResend)
		}

	default:
		return fmt.Errorf("invalid email transport method: %q", cfg.EmailTransport)
	}

	if len(cfg.EmailFrom) == 0 {
		return errors.New("email sender address is required")
	}

	return nil
}
