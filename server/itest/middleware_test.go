//go:build integration

package itest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/api"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
)

func TestAuthMiddleware_Success(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	s := user.Service{DB: testenv.DB()}
	s.Now = func() time.Time { return mustTimeUTC("2026-07-01 09:00:00") }
	aliceID, token := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a",
		mustTimeUTC("2026-07-01 09:00:00"),
	)

	var gotCalled bool
	var gotUID user.UserID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCalled = true
		gotUID, _ = api.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	req.Header.Set("Authorization", "Bearer "+token.Encode())
	rec := httptest.NewRecorder()
	api.AuthMiddleware(next, &s).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if !gotCalled {
		t.Error("next handler was not called")
	}
	if gotUID != aliceID {
		t.Errorf("context UserID=%d, want %d", gotUID, aliceID)
	}
}

func TestAuthMiddleware_Failure(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	s := user.Service{DB: testenv.DB()}
	_, token := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a",
		mustTimeUTC("2026-07-01 09:00:00"),
	)

	// Seed a second user whose token has already expired by the time the
	// middleware checks it (tokens expire 30 days after issuance).
	_, expiredToken := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "iPhone17",
		mustTimeUTC("2020-01-01 00:00:00"),
	)

	test := []struct {
		name       string
		authHeader string
	}{
		{name: "missing header", authHeader: ""},
		{name: "missing Bearer prefix", authHeader: token.Encode()},
		{name: "garbage token", authHeader: "Bearer not-a-real-token"},
		{name: "expired token", authHeader: "Bearer " + expiredToken.Encode()},
	}

	s.Now = func() time.Time { return mustTimeUTC("2026-07-01 09:01:00") }
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			var gotCalled bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/anything", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			api.AuthMiddleware(next, &s).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if gotCalled {
				t.Error("next handler must not be called")
			}
		})
	}
}
