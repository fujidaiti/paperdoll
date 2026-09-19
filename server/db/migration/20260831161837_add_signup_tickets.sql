-- +goose Up
CREATE TABLE signup_tickets (
    id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL,
    password_hash bytea NOT NULL,
    verification_code_hash bytea NOT NULL,
    ticket_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    fail_count int NOT NULL DEFAULT 0,
    issued_at timestamptz NOT NULL
);

CREATE INDEX signup_tickets_email_issued_at_idx
    ON signup_tickets (email, issued_at);

-- +goose Down
DROP TABLE signup_tickets;
