package main

import (
	"context"
	"database/sql"
)

// seedAuthSuit_NoUsers leaves the users table empty. It exists so the runner
// still starts a clean API server for scenarios that begin signed-out with no
// pre-existing accounts (sign-up, navigation).
func seedAuthSuit_NoUsers(ctx context.Context, db *sql.DB) error {
	return nil
}

// seedAuthSuit_SignedIn provisions the fixed test account so /signin can
// authenticate it, without seeding any other data. Used by scenarios that
// pump the app already authenticated but don't exercise feature-specific data
// (e.g. the sign-out test).
func seedAuthSuit_SignedIn(ctx context.Context, db *sql.DB) error {
	_, err := provisionTestAccount(ctx, db)
	return err
}

// seedAuthSuit_ExistingUser inserts a single account so sign-in and
// taken-email scenarios have something to authenticate against.
func seedAuthSuit_ExistingUser(ctx context.Context, db *sql.DB) error {
	_, err := provisionTestAccount(ctx, db)
	return err
}
