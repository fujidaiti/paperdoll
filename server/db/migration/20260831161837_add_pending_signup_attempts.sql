-- +goose Up
CREATE TABLE pending_signup_attempts (
    id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL,
    password_hash bytea NOT NULL,
    verification_code_hash bytea NOT NULL,
    ticket_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    fail_count int NOT NULL DEFAULT 0,
    signed_up_at timestamptz NOT NULL
);

CREATE INDEX pending_signup_attempts_email_signed_up_at_idx
    ON pending_signup_attempts (email, signed_up_at);

-- +goose Down
DROP TABLE pending_signup_attempts;
