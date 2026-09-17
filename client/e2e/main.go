package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fujidaiti/paperdoll/server/api"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/infra"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"golang.org/x/sync/errgroup"
)

// This is an E2E testing runner, which consists of these four components:
//
//   - [testenv], which manages a test DB container that is shared across all test cases.
//   - client, which is the Dart-side testing code that is driven by the Patrol framework.
//   - [messageHandler], which is a tiny HTTP server that listens to [message]s from the client.
//   - [sessionManager], which launches test [session]s based on the received messages and manages their lifecycle.
//
// At the beginning of each test case, the client sends a message to the messageHandler
// to initiates a test session. The lifecycle of a session consists of the following steps:
//
//  1. seeds the DB based on the requested scenario ID.
//  2. spawns a new API server instance.
//  3. tells the client that the server and DB are ready.
//  4. after that, the Flutter app can communicate with the server via HTTP.
//
// There is no specific message something like "tear down this session".
// All resources for the previous session is wiped out before starting a new session,
// including the seeded data and server instance.
//
// The client is supposed to run tests in sequence. Sending multiple messages is not illegal,
// but testing sessions never run in parallel as sessionManager handles messages one by one.
//
// The sessionManager never stops even if errors occur while handling messages or during a test session.
// This is a design decision so that non-fatal errors don't terminate the entire testing process.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		if err := testenv.ShutDown(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		cancel()
	}()
	// TODO: make the stub server address configurable
	if err := testenv.SetUp(ctx, "127.0.0.1:8081"); err != nil {
		return err
	}

	// This channel is unbuffered because:
	//  - the receiver (the sessionManager) processes messages one by one, and
	//  - senders (the messageHandler's handler functions) must wait for it to finish
	// 	  to tell the client if test sessions are successfully launched.
	msgc := make(chan message)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return messageHandler(ctx, msgc) })
	g.Go(func() error { sessionManager(ctx, msgc); return nil })
	g.Go(func() error {
		// stop() must be called here, otherwise a deadlock occurs in a happy path:
		//   1. all tests pass, runTests returns with no error.
		//   2. if messageHandler doesn't get any error, it doesn't return unless ctx is canceled.
		//   3. sessionManager never returns unless ctx is canceled.
		//   4. ctx is never canceled, as no goroutine returns an error.
		//   5. g.Wait() can't return until those two goroutines finish.
		defer stop()
		return runTests(ctx, patrolArgs())
	})
	return g.Wait()
}

// patrolArgs returns the extra arguments to forward to the underlying
// `patrol test` invocation, taken from everything after a literal "--" in
// the runner's own command-line arguments. This lets callers narrow down
// which tests run, e.g. to debug a single failing case:
//
//	go run ./e2e -- --tags smoke --target e2e/auth_test.dart
//
// See "patrol test --help" for the full set of options patrol accepts here.
func patrolArgs() []string {
	args := os.Args[1:]
	for i, a := range args {
		if a == "--" {
			return args[i+1:]
		}
	}
	return nil
}

// runTests drives the Patrol test suite via the `patrol test` CLI.
// extraArgs, if non-empty, is appended after the default arguments below.
func runTests(ctx context.Context, extraArgs []string) error {
	args := []string{
		"test",
		"--flutter-command",
		// TODO: make flutter command path configurable
		"../.fvm/flutter_sdk/bin/flutter",
		"-d",
		// TODO: make the target device configurable
		"emulator-5554",
		// TODO: make the API base URL configurable.
		"--dart-define",
		"API_BASE_URL=http://10.0.2.2:8080",
	}
	args = append(args, extraArgs...)
	cmd := exec.CommandContext(ctx, "patrol", args...)
	// The runner lives in client/e2e/, so the client package root is one level up.
	cmd.Dir = ".."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("testing failed with an error: %w", err)
	}
	return nil
}

type message struct {
	body    messageBody
	resultc chan<- string
}

type messageBody struct {
	SeederID string `json:"seeder_id"`
}

// lastSentEmail is the last email sent by the API server during the current session.
// This must be cleaned up at the beginning each session.
var lastSentEmail atomic.Pointer[infra.EmailDraft]

func messageHandler(ctx context.Context, msgc chan<- message) error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /setup", func(w http.ResponseWriter, r *http.Request) {
		var body messageBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			_, _ = fmt.Fprintf(w, "malformed message body: %v", err)
			return
		}
		resc := make(chan string, 1)
		select {
		case msgc <- message{body, resc}:
		case <-r.Context().Done():
			_, _ = fmt.Fprint(w, "request canceled")
			return
		}
		// Wait for the result to be returned.
		select {
		case res := <-resc:
			_, _ = fmt.Fprint(w, res)
		case <-r.Context().Done():
			_, _ = fmt.Fprint(w, "request canceled")
		}
	})

	// Signs in to the fixed test account with the [testAccountEmail] and returns
	// its bearer token. The seeder for the running session must have already
	// provisioned the account via provisionTestAccount; otherwise this fails.
	mux.HandleFunc("POST /signin", func(w http.ResponseWriter, r *http.Request) {
		email := must(user.ParseEmail(testAccountEmail))
		svc := &user.Service{DB: testenv.DB(), Now: time.Now}
		token, err := svc.SignIn(ctx, email, testAccountPassword, "TestDevice/1.0")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to sign in test account: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token.Encode()})
	})

	// Hands back the last email the running session sent to an address.
	mux.HandleFunc("GET /mailbox/last", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("addr")
		if addr == "" {
			http.Error(w, "the addr query parameter is required", http.StatusBadRequest)
			return
		}
		email := lastSentEmail.Load()
		if email == nil || email.To != addr {
			http.Error(w, fmt.Sprintf("no email has been sent to %s", addr), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, email.Body)
	})

	srv := http.Server{
		// TODO: make the host ip and port number configurable
		Addr:         ":9000",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}
	defer func() {
		sctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			fmt.Fprintf(os.Stderr, "got error while closing server: %v\n", err)
		}
	}()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "server exited abnormally: %v\n", err)
		}
	case <-ctx.Done():
	}
	return nil
}

func sessionManager(ctx context.Context, msgc <-chan message) {
	tearDown := func() { /* no-op */ }
	for {
		select {
		case msg := <-msgc:
			tearDown()
			done := make(chan struct{})
			sctx, cancel := context.WithCancel(ctx)
			tearDown = func() { cancel(); <-done }
			go session(sctx, done, msg)

		case <-ctx.Done():
			tearDown()
			return
		}
	}
}

func session(ctx context.Context, done chan struct{}, msg message) {
	defer close(done)
	defer testenv.TearDown()
	if err := seedDB(ctx, testenv.DB(), msg.body.SeederID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to seed DB for scenario %q: %v", msg.body.SeederID, err)
		// TODO: return a better response message
		msg.resultc <- "failed to seed DB"
		return
	}

	lastSentEmail.Store(nil)
	emailSender := func(d infra.EmailDraft) error {
		lastSentEmail.Store(&d)
		return nil
	}

	// TODO: make stub HTTP server address configurable
	proxyURL, _ := url.Parse("http://127.0.0.1:8081")
	srv := api.NewServer(testenv.DB(), proxyURL, emailSender)
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown failed: %v\n", err)
		}
	}()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		msg.resultc <- fmt.Sprintf("failed to start server: %v", err)
		return
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	// TODO: return a cleaner response
	msg.resultc <- "ready"

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "server exited abnormally: %v\n", err)
		}

	case <-ctx.Done():
	}
}
