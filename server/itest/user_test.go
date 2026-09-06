//go:build integration

package itest

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/infra"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"github.com/google/go-cmp/cmp"
)

type userRecord struct {
	ID           int
	Email        string
	PasswordHash []byte
}

type authTokenRecord struct {
	ID        int
	UserId    int
	Device    string
	TokenHash []byte
	ExpiresAt time.Time
}

type pendingSignUpAttemptRecord struct {
	ID                   int
	Email                string
	PasswordHash         []byte
	VerificationCodeHash []byte
	TicketHash           []byte
	ExpiresAt            time.Time
}

func TestAuth_SignUp_Success(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	test := []struct {
		name     string
		email    string
		password string
		code     string
		signUpAt time.Time
	}{
		{
			name:     "alice",
			email:    "alice@gmail.com",
			password: "Test$Password+123",
			code:     "123456",
			signUpAt: mustTimeUTC("2026-07-01 09:15:00"),
		},
		{
			name:     "alice (the second attempt)",
			email:    "alice@gmail.com",
			password: "New$Password+456",
			code:     "654321",
			signUpAt: mustTimeUTC("2026-07-01 09:20:00"),
		},
		{
			name:     "bob (same password as alice)",
			email:    "bob@exchange.com",
			password: "Test$Password+123",
			code:     "162534",
			signUpAt: mustTimeUTC("2027-08-20 14:00:59"),
		},
	}

	var gotPswdHashes []string
	var gotTickets []user.Token
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			var gotEmailDraft infra.Draft
			captureEmailSent := func(d infra.Draft) error {
				gotEmailDraft = d
				return nil
			}

			gotTicket, gotErr := user.SignUp(
				t.Context(),
				must(user.ParseEmail(tt.email)),
				must(user.ValidatePassword(tt.password)),
				testenv.DB(),
				tt.signUpAt,
				func() (user.VerificationCode, error) {
					return user.VerificationCode(tt.code), nil
				},
				captureEmailSent,
			)
			if gotErr != nil {
				t.Fatalf("got %q, want nil error", gotErr)
			}
			gotTickets = append(gotTickets, gotTicket)

			var gotAtmpt pendingSignUpAttemptRecord
			scanRowOrFatal(t, `
				SELECT id, email, password_hash, verification_code_hash, ticket_hash, expires_at
				FROM pending_signup_attempts WHERE email = $1
				ORDER BY signed_up_at DESC LIMIT 1
			`, []any{tt.email}, &gotAtmpt.ID, &gotAtmpt.Email, &gotAtmpt.PasswordHash,
				&gotAtmpt.VerificationCodeHash, &gotAtmpt.TicketHash, &gotAtmpt.ExpiresAt,
			)
			gotPswdHashes = append(gotPswdHashes, string(gotAtmpt.PasswordHash))

			if gotAtmpt.Email != tt.email {
				t.Errorf("got %q, want %q", gotAtmpt.Email, tt.email)
			}
			if bytes.Equal(gotAtmpt.PasswordHash, []byte(tt.password)) {
				t.Error("raw password must not be stored")
			}
			if bytes.Equal(gotAtmpt.VerificationCodeHash, []byte(tt.code)) {
				t.Error("raw code must not be stored")
			}
			if bytes.Equal(gotAtmpt.TicketHash, []byte(gotTicket.Encode())) {
				t.Errorf("raw ticket must not be stored")
			}
			if d := gotAtmpt.ExpiresAt.Sub(tt.signUpAt); d != 10*time.Minute {
				t.Errorf(
					"got TTL %g min, want 10 min; expiresAt: %s, signUpAt:%s",
					d.Minutes(),
					gotAtmpt.ExpiresAt,
					tt.signUpAt,
				)
			}

			if gotEmailDraft.To != tt.email {
				t.Errorf("got email recipient %q, want %q", gotEmailDraft.To, tt.email)
			}
			if gotEmailDraft.Subject == "" {
				t.Errorf("email subject must not be empty")
			}
			codeRe := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(tt.code)))
			if n := len(codeRe.FindAllString(gotEmailDraft.Body, 2)); n == 0 {
				t.Error("no code found in email body")
			} else if n > 1 {
				t.Error("found multiple codes in email body, want exactly one")
			}
		})
	}

	if !isDistinct(gotPswdHashes) {
		t.Errorf("password hashes must be uniqueue even if raw passwords are identical")
	}
	if !isDistinct(gotTickets) {
		t.Errorf("tickets must be uniqueue")
	}
}

func TestAuth_SignUp_EmailUniqueness(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	var (
		aliceAddr = "alice@example.com"
		alicePswd = must(user.ValidatePassword("Test$Password123"))
	)
	var alice userRecord
	scanRowOrFatal(t, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, password_hash
	`, []any{aliceAddr, must(alicePswd.Hash())}, &alice.ID, &alice.Email, &alice.PasswordHash)

	test := []struct {
		name     string
		email    string
		password user.ValidPassword
		signUpAt time.Time
	}{
		{
			name:     "same email and different password",
			email:    aliceAddr,
			password: must(user.ValidatePassword("test$Password987")),
			signUpAt: mustTimeUTC("2026-09-05 20:34:00"),
		},
		{
			name:     "same email and same password",
			email:    aliceAddr,
			password: alicePswd,
			signUpAt: mustTimeUTC("2026-09-05 20:35:00"),
		},
		{
			name:     "same but capitalized email",
			email:    "ALICE@EXAMPLE.COM",
			password: alicePswd,
			signUpAt: mustTimeUTC("2026-09-05 20:36:00"),
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			_, got := user.SignUp(
				t.Context(),
				must(user.ParseEmail(tt.email)),
				tt.password,
				testenv.DB(),
				tt.signUpAt,
				user.NewVerificationCode,
				func(_ infra.Draft) error { return nil },
			)
			if want := user.ErrEmailTaken; !errors.Is(got, want) {
				t.Errorf("got %q, want %q", got, want)
			}

			gotUsers := scanRowsOrFatal(t, `
				SELECT id, email, password_hash FROM users
			`, nil, func(r *sql.Rows, d *userRecord) error {
				return r.Scan(&d.ID, &d.Email, &d.PasswordHash)
			})

			if n := len(gotUsers); n != 1 {
				t.Fatalf("exactly one user must be registered, got %d users", n)
			}
			if d := cmp.Diff(gotUsers[0], alice); d != "" {
				t.Errorf("already registered user must never be touched, diff:\n%s", d)
			}
		})
	}
}

func TestAuth_SignUp_PerAddressThrottle(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	test := []struct {
		signUpAt time.Time
		wantErr  error
	}{
		// Users can try three times per address within an hour.
		{mustTimeUTC("2026-09-03 12:00:00"), nil},
		{mustTimeUTC("2026-09-03 12:10:58"), nil},
		{mustTimeUTC("2026-09-03 12:30:00"), nil},
		{mustTimeUTC("2026-09-03 12:30:01"), user.ErrTooManyAttempts},
		{mustTimeUTC("2026-09-03 13:00:00"), user.ErrTooManyAttempts},
		{mustTimeUTC("2026-09-03 13:00:01"), nil},
		{mustTimeUTC("2026-09-03 13:10:58"), user.ErrTooManyAttempts},
		{mustTimeUTC("2026-09-03 13:10:59"), nil},
		{mustTimeUTC("2026-09-03 14:11:00"), nil},
		{mustTimeUTC("2026-09-03 14:11:30"), nil},
		{mustTimeUTC("2026-09-03 14:12:00"), nil},
		{mustTimeUTC("2026-09-03 14:12:30"), user.ErrTooManyAttempts},
	}

	for i, tt := range test {
		t.Run(fmt.Sprintf("attempt %d", i+1), func(t *testing.T) {
			_, got := user.SignUp(
				t.Context(),
				must(user.ParseEmail("alice@example.com")),
				must(user.ValidatePassword("Test$Password#1234")),
				testenv.DB(),
				tt.signUpAt,
				func() (user.VerificationCode, error) { return "123456", nil },
				func(_ infra.Draft) error { return nil },
			)
			if !errors.Is(got, tt.wantErr) {
				t.Errorf("got %q, want %q", got, tt.wantErr)
			}
		})
	}
}

func TestAuth_SignIn_Success(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	type User struct {
		email, password, signUpDevice string
		signUpAt                      time.Time
	}
	users := map[string]*User{
		"alice": {
			email:        "alice@example.com",
			password:     "alice#password$123",
			signUpDevice: "Pixel9a/Android17",
			signUpAt:     mustTimeUTC("2026-07-01 09:15:00"),
		},
		"bob": {
			email:        "bob@forest.com",
			password:     "bob#password$123",
			signUpDevice: "GalaxyS26/Android16",
			signUpAt:     mustTimeUTC("2026-07-10 14:40:05"),
		},
	}
	test := []struct {
		name, device string
		user         *User
		signInAt     time.Time
	}{
		{
			name:     "alice's first session",
			device:   "Pixel9a/Android17",
			user:     users["alice"],
			signInAt: mustTimeUTC("2026-07-01 09:15:30"),
		},
		{
			name:     "bob's first session",
			device:   "GalaxyS26/Android16",
			user:     users["bob"],
			signInAt: mustTimeUTC("2026-07-10 14:45:18"),
		},
		{
			name:     "alice's second session",
			device:   "Pixel9a/Android17",
			user:     users["alice"],
			signInAt: mustTimeUTC("2026-07-08 07:45:00"),
		},
		{
			name:     "alice's third session from different device",
			device:   "iPhone17/iOS26",
			user:     users["alice"],
			signInAt: mustTimeUTC("2026-07-14 21:05:40"),
		},
		{
			name:   "bob's second session but email capitalized",
			device: "GalaxyS26/Android16",
			user: &User{
				email:        "BOB@FOREST.COM",
				password:     users["bob"].password,
				signUpDevice: users["bob"].signUpDevice,
				signUpAt:     users["bob"].signUpAt,
			},
			signInAt: mustTimeUTC("2026-07-10 14:45:18"),
		},
	}

	s := user.Service{DB: testenv.DB()}
	// Seed users
	for _, u := range users {
		provisionTestAccount(t, u.email, u.password, u.signUpDevice, u.signUpAt)
	}

	var gotTokens []user.Token
	for i, tt := range test {
		s.Now = func() time.Time { return tt.signInAt }
		t.Run(tt.name, func(t *testing.T) {
			email := must(user.ParseEmail(tt.user.email))
			gotToken, err := s.SignIn(t.Context(), email, tt.user.password, tt.device)
			if err != nil {
				t.Fatalf("failed to sign-in: %v", err)
			}
			gotTokens = append(gotTokens, gotToken)

			var n int
			scanRowOrFatal(t, `SELECT COUNT(*) from auth_tokens`, nil, &n)
			if want := len(users) + i + 1; want != n {
				t.Errorf("only one token record must be added, got %d extra rows", n-want)
			}

			var gotRec authTokenRecord
			scanRowOrFatal(t, `
				SELECT user_id, device, token_hash, expires_at FROM auth_tokens
				ORDER BY created_at DESC LIMIT 1
			`, nil, &gotRec.UserId, &gotRec.Device, &gotRec.TokenHash, &gotRec.ExpiresAt)

			var gotEmail string
			scanRowOrFatal(t, `
				SELECT email FROM users WHERE id = $1
			`, []any{gotRec.UserId}, &gotEmail)
			if got, err := user.ParseEmail(gotEmail); err != nil {
				t.Errorf("saved email %q is malformed, want %v", gotEmail, email)
			} else if got != email {
				t.Errorf("token was issued for wrong user %v, want %v", got, email)
			}

			if gotRec.Device != tt.device {
				t.Errorf("got device %q, want %q", gotRec.Device, tt.device)
			}

			if d := gotRec.ExpiresAt.Sub(tt.signInAt); d != 30*24*time.Hour {
				t.Errorf("token should expires in 30 days, actual TTL is %g day(s)", d.Hours()/24)
			}

			if d := cmp.Diff(gotRec.TokenHash, gotToken.Hash()); d != "" {
				t.Errorf("token must be hashed in DB, diff:\n%s", d)
			}
		})
	}

	if !isDistinct(gotTokens) {
		t.Errorf("tokens must be uniqueue across all sessions")
	}
}

func TestAuth_SignIn_Failure(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	alice := struct {
		email, password, signUpDevice string
		signUpAt                      time.Time
	}{
		"alice@example.com", "alice#password$123", "Pixel9a/Android16",
		mustTimeUTC("2026-07-01 09:15:00"),
	}
	test := []struct {
		name, email, password, device string
		signedInAt                    time.Time
		wantErr                       error
	}{
		{
			name:       "wrong password",
			email:      alice.email,
			password:   "wrong#" + alice.password,
			device:     alice.signUpDevice,
			signedInAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrAuthFailed,
		},
		{
			name:       "unregistered user",
			email:      "unregistered." + alice.email,
			password:   alice.password,
			device:     alice.signUpDevice,
			signedInAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrAuthFailed,
		},
		{
			name:       "no device info",
			email:      alice.email,
			password:   alice.password,
			device:     "",
			signedInAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrDeviceEmpty,
		},
	}

	// Seed user
	s := user.Service{DB: testenv.DB()}
	provisionTestAccount(t, alice.email, alice.password, alice.signUpDevice, alice.signUpAt)

	for _, tt := range test {
		s.Now = func() time.Time { return tt.signedInAt }
		t.Run(tt.name, func(t *testing.T) {
			got1, got2 := s.SignIn(
				t.Context(), must(user.ParseEmail(tt.email)),
				tt.password, tt.device,
			)
			if !errors.Is(got2, tt.wantErr) {
				t.Errorf("got %v, want %v", got2, tt.wantErr)
			}
			if got := got1.Encode(); got != "" {
				t.Errorf("must be an empty token, got %v", got)
			}

			var n int
			scanRowOrFatal(t, `SELECT COUNT(*) from auth_tokens`, nil, &n)
			if got := n - 1; got != 0 {
				t.Errorf("no extra token must be issued, got %d extra rows", got)
			}
		})
	}
}

func TestAuth_SignOut(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	s := user.Service{DB: testenv.DB()}
	// Sign up
	addr, pswd := "alice@example.com", "alice#password$123"
	_, token1 := provisionTestAccount(t, addr, pswd, "Pixel9a", mustTimeUTC("2026-07-01 13:30:00"))

	// Sign in from other devices
	email := must(user.ParseEmail(addr))
	s.Now = func() time.Time { return mustTimeUTC("2026-07-02 23:18:45") }
	token2 := must(s.SignIn(t.Context(), email, pswd, "iPhone17"))

	s.Now = func() time.Time { return mustTimeUTC("2026-07-04 19:30:00") }
	token3 := must(s.SignIn(t.Context(), email, pswd, "macbookAir2020"))

	test := []struct {
		name        string
		token       user.Token
		signedOutAt time.Time
	}{
		{
			name:        "from signed-up device",
			token:       token1,
			signedOutAt: mustTimeUTC("2026-07-08 12:33:33"),
		},
		{
			name:        "from signed-in device",
			token:       token2,
			signedOutAt: mustTimeUTC("2026-07-12 02:10:00"),
		},
		{
			name:        "already signed out",
			token:       token2,
			signedOutAt: mustTimeUTC("2026-07-12 02:11:00"),
		},
		{
			name:        "unregistered user",
			token:       user.Token{},
			signedOutAt: mustTimeUTC("2026-08-01 09:00:00"),
		},
		{
			name:        "outdated",
			token:       token3,
			signedOutAt: mustTimeUTC("2029-11-04 12:00:00"),
		},
	}

	for _, tt := range test {
		s.Now = func() time.Time { return tt.signedOutAt }
		t.Run(tt.name, func(t *testing.T) {
			if got := s.SignOut(t.Context(), tt.token.Encode()); got != nil {
				t.Errorf("got %v, want a nil error", got)
			}

			var n int
			scanRowOrFatal(t, `
				SELECT COUNT(*) FROM auth_tokens WHERE token_hash = $1
			`, []any{tt.token.Hash()}, &n)
			if n != 0 {
				t.Errorf("got %d rows: token still exists", n)
			}
		})
	}
}

func TestAuth_VerifyAuthToken(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	s := user.Service{DB: testenv.DB()}
	aEmail, aPswd := "alice@example.com", "alice#password$123"
	bEmail, bPswd := "bob@example.com", "bob#password$789"
	// Alice signs up
	aID, token1 := provisionTestAccount(t,
		aEmail, aPswd, "Pixel9a", mustTimeUTC("2026-07-01 13:30:00"))
	// Alice signs in from another device
	s.Now = func() time.Time { return mustTimeUTC("2026-07-02 09:00:00") }
	token2, err := s.SignIn(t.Context(), must(user.ParseEmail(aEmail)), aPswd, "iPad")
	if err != nil {
		t.Fatalf("failed to sign in: %v", err)
	}
	// Bob signs up
	bID, token3 := provisionTestAccount(t,
		bEmail, bPswd, "iPhone17", mustTimeUTC("2026-07-02 23:18:45"))
	// Bob signs in from another device
	s.Now = func() time.Time { return mustTimeUTC("2026-07-04 19:30:00") }
	token4, err := s.SignIn(t.Context(), must(user.ParseEmail(bEmail)), bPswd, "macbookAir 2020")
	if err != nil {
		t.Fatalf("failed to sign in: %v", err)
	}

	test := []struct {
		name    string
		token   user.Token
		checkAt time.Time
		want    user.UserID
		wantErr error
	}{
		{
			name:    "alice's sign-up token",
			token:   token1,
			checkAt: mustTimeUTC("2026-07-01 13:30:30"),
			want:    aID,
		},
		{
			name:    "alice's sign-in token",
			token:   token2,
			checkAt: mustTimeUTC("2026-07-02 09:12:32"),
			want:    aID,
		},
		{
			name:    "bob's sign-up token",
			token:   token3,
			checkAt: mustTimeUTC("2026-07-12 02:10:00"),
			want:    bID,
		},
		{
			name:    "bob's expired sign-in token",
			token:   token4,
			checkAt: mustTimeUTC("2026-11-04 12:00:00"),
			want:    0,
			wantErr: user.ErrTokenInvalid,
		},
		{
			name:    "unknown token",
			token:   user.Token{},
			checkAt: mustTimeUTC("2026-08-01 09:00:00"),
			want:    0,
			wantErr: user.ErrTokenInvalid,
		},
	}

	for _, tt := range test {
		s.Now = func() time.Time { return tt.checkAt }
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotErr := s.VerifyAuthToken(t.Context(), tt.token.Encode())
			if gotID != tt.want {
				t.Errorf("got ID=%d, want %d", gotID, tt.want)
			}
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got %q, want %q", gotErr, tt.wantErr)
			}
		})
	}
}

func TestAuth_VerifySignUpEmailAddress_Success(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	test := []struct {
		name, email, password, code, device string
		signUpAt, verifiedAt                time.Time
	}{
		{
			name:       "alice",
			email:      "alice@gmail.com",
			password:   "Test$Password+123",
			code:       "123456",
			device:     "Pixel9a/Android16",
			signUpAt:   mustTimeUTC("2026-07-01 09:20:00"),
			verifiedAt: mustTimeUTC("2026-07-01 09:25:00"),
		},
		{
			name:       "bob (same password and code as alice)",
			email:      "bob@exchange.com",
			password:   "Test$Password+123",
			code:       "123456",
			device:     "iPhone17/iOS26",
			signUpAt:   mustTimeUTC("2027-08-20 14:05:00"),
			verifiedAt: mustTimeUTC("2027-08-20 14:10:00"),
		},
	}

	s := user.Service{DB: testenv.DB()}
	var gotTokens []user.Token
	for i, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			ticket := must(user.SignUp(
				t.Context(),
				must(user.ParseEmail(tt.email)),
				must(user.ValidatePassword(tt.password)),
				testenv.DB(),
				tt.signUpAt,
				func() (user.VerificationCode, error) { return user.VerificationCode(tt.code), nil },
				func(infra.Draft) error { return nil },
			))
			s.Now = func() time.Time { return tt.verifiedAt }
			gotToken, gotErr := s.VerifySignUpEmailAddress(t.Context(), ticket.Encode(), tt.code, tt.device)
			if gotErr != nil {
				t.Fatalf("failed to verify the email address: %v", gotErr)
			}
			if gotToken.Encode() == "" {
				t.Fatal("an auth token must be issued on success")
			}
			gotTokens = append(gotTokens, gotToken)

			var gotUser userRecord
			scanRowOrFatal(t, `
				SELECT id, email, password_hash FROM users WHERE email = $1
			`, []any{tt.email}, &gotUser.ID, &gotUser.Email, &gotUser.PasswordHash)

			if gotUser.Email != tt.email {
				t.Errorf("created user has a malformed email %q, want %q", gotUser.Email, tt.email)
			}

			wantPswdHash := scanValOrFatal[[]byte](t, `
				SELECT password_hash FROM pending_signup_attempts WHERE email = $1
			`, tt.email)
			if d := cmp.Diff(gotUser.PasswordHash, wantPswdHash); d != "" {
				t.Errorf("password hash must be carried over from the attempt, diff:\n%s", d)
			}

			var n int
			scanRowOrFatal(t, `SELECT COUNT(*) from auth_tokens`, nil, &n)
			if want := i + 1; n != want {
				t.Errorf("only one token record must be added, got %d extra row(s)", n-want)
			}

			var gotTkn authTokenRecord
			scanRowOrFatal(t, `
				SELECT user_id, device, token_hash, expires_at FROM auth_tokens
				ORDER BY created_at DESC LIMIT 1
			`, nil, &gotTkn.UserId, &gotTkn.Device, &gotTkn.TokenHash, &gotTkn.ExpiresAt)

			if got, want := gotTkn.UserId, gotUser.ID; got != want {
				t.Errorf("token was issued for wrong user Id=%d, want Id=%d", got, want)
			}
			if gotTkn.Device != tt.device {
				t.Errorf("got device %q, want %q", gotTkn.Device, tt.device)
			}
			if d := gotTkn.ExpiresAt.Sub(tt.verifiedAt); d != 30*24*time.Hour {
				t.Errorf("token should expires in 30 days, got TTL = %g day(s)", d.Hours()/24)
			}
			if bytes.Equal(gotTkn.TokenHash, []byte(gotToken.Encode())) {
				t.Error("raw token must not be stored")
			}
		})
	}

	if !isDistinct(gotTokens) {
		t.Errorf("auth tokens must be uniqueue")
	}
}

func TestAuth_VerifySignUpEmailAddress_Failure(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	alice := struct {
		email, password, code, device string
		signUpAt                      time.Time
	}{
		"alice@example.com", "alice#password$123", "123456", "Pixel9a/Android16",
		mustTimeUTC("2026-07-01 09:15:00"),
	}

	aliceTicket := must(user.SignUp(
		t.Context(),
		must(user.ParseEmail(alice.email)),
		must(user.ValidatePassword(alice.password)),
		testenv.DB(),
		alice.signUpAt,
		func() (user.VerificationCode, error) { return user.VerificationCode(alice.code), nil },
		func(infra.Draft) error { return nil },
	)).Encode()

	test := []struct {
		name, ticket, code, device string
		signUpAt, verifiedAt       time.Time
		wantErr                    error
	}{
		{
			name:       "wrong code",
			ticket:     aliceTicket,
			code:       "654321",
			device:     alice.device,
			signUpAt:   mustTimeUTC("2026-07-01 09:15:00"),
			verifiedAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrEmailVerifyFailed,
		},
		{
			name:       "unknown ticket",
			ticket:     must(user.DecodeToken("tLTHpBZIggBuaRO_TGmz0MQkrlQ4tsjXnwFMIQINRnY")).Encode(),
			code:       alice.code,
			device:     alice.device,
			signUpAt:   mustTimeUTC("2026-07-01 09:15:00"),
			verifiedAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrEmailVerifyFailed,
		},
		{
			name:       "malformed ticket",
			ticket:     "invalid",
			code:       alice.code,
			device:     alice.device,
			signUpAt:   mustTimeUTC("2026-07-01 09:15:00"),
			verifiedAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrEmailVerifyFailed,
		},
		{
			name:       "no device info",
			ticket:     aliceTicket,
			code:       alice.code,
			device:     "",
			signUpAt:   mustTimeUTC("2026-07-01 09:15:00"),
			verifiedAt: mustTimeUTC("2026-07-01 09:16:00"),
			wantErr:    user.ErrDeviceEmpty,
		},
	}

	s := user.Service{DB: testenv.DB()}
	for _, tt := range test {
		fmt.Printf("ticket: %s", tt.ticket)
		t.Run(tt.name, func(t *testing.T) {
			s.Now = func() time.Time { return tt.verifiedAt }
			gotToken, gotErr := s.VerifySignUpEmailAddress(t.Context(), tt.ticket, tt.code, tt.device)

			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got %q, want %q", gotErr, tt.wantErr)
			}
			if got := gotToken.Encode(); got != "" {
				t.Errorf("must be an empty token, got %v", got)
			}

			var nUsers int
			scanRowOrFatal(t, `SELECT COUNT(*) FROM users`, nil, &nUsers)
			if nUsers != 0 {
				t.Errorf("no user must be created, got %d rows", nUsers)
			}

			var nTokens int
			scanRowOrFatal(t, `SELECT COUNT(*) FROM auth_tokens`, nil, &nTokens)
			if nTokens != 0 {
				t.Errorf("no token must be issued, got %d rows", nTokens)
			}
		})
	}
}

func TestAuth_VerifySignUpEmailAddress_DuplicateVerifications(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	const (
		email  = "alice@example.com"
		code   = "123456"
		device = "Pixel9a/Android"
	)
	ticket := must(user.SignUp(
		t.Context(),
		must(user.ParseEmail(email)),
		must(user.ValidatePassword("Test#Password$1234")),
		testenv.DB(), mustTimeUTC("2026-09-03 11:50:00"),
		func() (user.VerificationCode, error) { return user.VerificationCode(code), nil },
		func(infra.Draft) error { return nil },
	)).Encode()

	s := user.Service{
		DB:  testenv.DB(),
		Now: func() time.Time { return mustTimeUTC("2026-09-03 11:52:00") },
	}
	_ = must(s.VerifySignUpEmailAddress(t.Context(), ticket, code, device))

	s.Now = func() time.Time { return mustTimeUTC("2026-09-03 11:58:00") }
	_, gotErr := s.VerifySignUpEmailAddress(t.Context(), ticket, code, device)
	if want := user.ErrEmailTaken; !errors.Is(gotErr, want) {
		t.Errorf(
			"Duplicate verifications must fail even the attempt isn't expired and the code is correct: "+
				"got %q, want %q", gotErr, want,
		)
	}

	var n int
	scanRowOrFatal(t, `SELECT COUNT(*) FROM users`, []any{}, &n)
	if n != 1 {
		t.Errorf("got %d user rows, want exactly one", n)
	}
	scanRowOrFatal(t, `SELECT COUNT(*) FROM auth_tokens`, []any{}, &n)
	if n != 1 {
		t.Errorf("got %d token rows, want exactly one", n)
	}
}

func TestAuth_VerifySignUpEmailAddress_TicketExpiry(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	// SignUp gives each attempt a 10-minute lifetime.
	signUpAt := mustTimeUTC("2026-07-01 09:20:00")
	expiresAt := mustTimeUTC("2026-07-01 09:30:00")
	test := []struct {
		name        string
		email       string
		verifyAt    time.Time
		wantErr     error
		wantAccount bool
	}{
		{
			name:        "before the expiry",
			email:       "alice@example.com",
			verifyAt:    expiresAt.Add(-time.Second),
			wantErr:     nil,
			wantAccount: true,
		},
		{
			name:        "exactly at the expiry",
			email:       "bob@example.com",
			verifyAt:    expiresAt,
			wantErr:     nil,
			wantAccount: true,
		},
		{
			name:        "after the expiry",
			email:       "carol@example.com",
			verifyAt:    expiresAt.Add(time.Second),
			wantErr:     user.ErrEmailVerifyCodeExpired,
			wantAccount: false,
		},
	}

	s := user.Service{DB: testenv.DB()}
	for _, tt := range test {
		ticket := must(user.SignUp(
			t.Context(),
			must(user.ParseEmail(tt.email)),
			must(user.ValidatePassword("test#password$123")),
			testenv.DB(), signUpAt,
			func() (user.VerificationCode, error) { return user.VerificationCode("123456"), nil },
			func(infra.Draft) error { return nil },
		))
		s.Now = func() time.Time { return tt.verifyAt }

		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotErr := s.VerifySignUpEmailAddress(
				t.Context(), ticket.Encode(), "123456", "Pixel9a/Android",
			)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("got %q, want %q", gotErr, tt.wantErr)
			}
			switch tkn := gotToken.Encode(); {
			case tt.wantAccount && tkn == "":
				t.Errorf("got empty token, want valid one")
			case !tt.wantAccount && tkn != "":
				t.Errorf("got %v, want empty token", gotToken)
			}

			var nUsers int
			scanRowOrFatal(t, `SELECT COUNT(*) FROM users WHERE email = $1`, []any{tt.email}, &nUsers)
			switch {
			case tt.wantAccount && nUsers != 1:
				t.Errorf("exactly one user must be created, got %d rows", nUsers)
			case !tt.wantAccount && nUsers != 0:
				t.Errorf("no user must be created, got %d rows", nUsers)
			}
		})
	}
}

func TestAuth_VerifySignUpEmailAddress_EmailAlreadyRegistered(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	const (
		email          = "alice@example.com"
		signUpPassword = "alice#password$123"
		signUpDevice   = "Pixel9a/Android16"
		// The pending attempt was started before the address got registered.
		pendingPassword = "alice#password$987"
		pendingCode     = "123456"
		pendingDevice   = "iPhone17/iOS26"
	)

	// The pending attempt is registered before the address gets taken.
	pendingTicket := must(user.SignUp(
		t.Context(),
		must(user.ParseEmail(email)),
		must(user.ValidatePassword(pendingPassword)),
		testenv.DB(),
		mustTimeUTC("2026-07-01 09:20:00"),
		func() (user.VerificationCode, error) { return user.VerificationCode(pendingCode), nil },
		func(infra.Draft) error { return nil },
	)).Encode()

	provisionTestAccount(
		t, email, signUpPassword, signUpDevice, mustTimeUTC("2026-07-01 09:00:00"),
	)

	var existingUser userRecord
	scanRowOrFatal(t, `
		SELECT id, email, password_hash FROM users WHERE email = $1
	`, []any{email}, &existingUser.ID, &existingUser.Email, &existingUser.PasswordHash)

	s := user.Service{
		DB:  testenv.DB(),
		Now: func() time.Time { return mustTimeUTC("2026-07-01 09:15:00") },
	}
	gotToken, gotErr := s.VerifySignUpEmailAddress(
		t.Context(), pendingTicket, pendingCode, pendingDevice)

	if gotErr == nil {
		t.Fatal("want an error, got nil")
	}
	if got := gotToken.Encode(); got != "" {
		t.Errorf("must be an empty token, got %v", got)
	}

	gotUsers := scanRowsOrFatal(t, `
		SELECT id, email, password_hash FROM users
	`, nil, func(r *sql.Rows, d *userRecord) error {
		return r.Scan(&d.ID, &d.Email, &d.PasswordHash)
	})
	if n := len(gotUsers); n != 1 {
		t.Fatalf("exactly one user must be registered, got %d users", n)
	}

	if d := cmp.Diff(gotUsers[0], existingUser); d != "" {
		t.Errorf("already registered user must never be touched, diff:\n%s", d)
	}

	var nTokens int
	scanRowOrFatal(t, `
		SELECT COUNT(*) FROM auth_tokens WHERE device = $1
	`, []any{pendingDevice}, &nTokens)
	if nTokens != 0 {
		t.Errorf("no token must be issued for the pending device, got %d rows", nTokens)
	}
}

func TestAuth_VerifySignUpEmailAddress_FailCountCap(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	code := "123456"
	ticket := must(user.SignUp(
		t.Context(),
		must(user.ParseEmail("alice@example.com")),
		must(user.ValidatePassword("alice#password$123")),
		testenv.DB(), mustTimeUTC("2026-07-01 09:20:00"),
		func() (user.VerificationCode, error) { return user.VerificationCode(code), nil },
		func(infra.Draft) error { return nil },
	))

	test := []struct {
		code     string
		wantErr  error
		verifyAt time.Time
	}{
		{"000000", user.ErrEmailVerifyFailed, mustTimeUTC("2026-07-01 09:20:00")},
		{"111111", user.ErrEmailVerifyFailed, mustTimeUTC("2026-07-01 09:20:20")},
		{"222222", user.ErrEmailVerifyFailed, mustTimeUTC("2026-07-01 09:20:40")},
		{"333333", user.ErrEmailVerifyFailed, mustTimeUTC("2026-07-01 09:21:00")},
		{"444444", user.ErrEmailVerifyFailed, mustTimeUTC("2026-07-01 09:21:30")},
		// The last wrong code reached the cap (5 times), so the ticket is dead:
		// even the correct code must not verify it.
		{code, user.ErrEmailVerifyCodeExpired, mustTimeUTC("2026-07-01 09:22:00")},
	}

	s := user.Service{DB: testenv.DB()}
	for i, tt := range test {
		attempt := i + 1
		s.Now = func() time.Time { return tt.verifyAt }
		t.Run(fmt.Sprintf("attempt %d", attempt), func(t *testing.T) {
			gotToken, gotErr := s.VerifySignUpEmailAddress(
				t.Context(), ticket.Encode(), tt.code, "Pixel9a/Android",
			)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Fatalf("attempt %d: got %q, want %q", attempt, gotErr, tt.wantErr)
			}
			if got := gotToken.Encode(); got != "" {
				t.Errorf("attempt %d: got %v, want an empty token", attempt, got)
			}
		})
	}

	var n int
	scanRowOrFatal(t, `SELECT COUNT(*) FROM users`, nil, &n)
	if n != 0 {
		t.Errorf("no user must be created, got %d rows", n)
	}
	scanRowOrFatal(t, `SELECT COUNT(*) FROM auth_tokens`, nil, &n)
	if n != 0 {
		t.Errorf("no token must be issued, got %d rows", n)
	}
}
