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
	//  - [EmailTransportDebug], which uses SMTP to communicate with a development
	//    mail server such as Mailpit. Do not use it in production as this method
	//    never cares about injection attacks.
	EmailTransport EmailTransport `env:"EMAIL_TRANSPORT,required"`

	// The From address used when sending an email.
	EmailFrom string `env:"EMAIL_FROM,required"`

	// The SMTP server host, such as localhost. Required when [Config.EmailTransport]
	// is [EmailTransportDebug].
	SMTPHost string `env:"SMTP_HOST"`

	// The SMTP server port, such as 1025. Required when [Config.EmailTransport] is
	// [EmailTransportDebug].
	SMTPPort string `env:"SMTP_PORT"`
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
	EmailTransportDebug EmailTransport = "dbg"
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

	default:
		return fmt.Errorf("invalid email transport method: %q", cfg.EmailTransport)
	}

	if len(cfg.EmailFrom) == 0 {
		return errors.New("email sender address is required")
	}

	return nil
}
