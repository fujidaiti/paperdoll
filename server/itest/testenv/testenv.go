package testenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/fujidaiti/paperdoll/server/db/migration"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type LaunchOption struct {
	// Whether starts a mail server so that tests can read the emails
	// the API server sends. It is off by default.
	EnableMailServer bool

	// The host IP address the mail server is bound to.
	// Required when [LaunchOption.EnableMailServer] is true.
	MailServerHost netip.Addr

	// The host port the SMTP listener is bound to.
	// Required when [LaunchOption.EnableMailServer] is true.
	MailServerSMTPPort string
}

// SetUp initializes a test container and migrate the database.
// Make sure to always call [ShutDown] even if this returns a non-nil error.
func SetUp(ctx context.Context, stubAddr string, opt LaunchOption) error {
	if dbServer != nil || db != nil {
		panic("do not call SetUp twice")
	}
	if err := startDBServer(ctx); err != nil {
		return err
	}
	if opt.EnableMailServer {
		if err := startMailServer(ctx, opt.MailServerHost, opt.MailServerSMTPPort); err != nil {
			return err
		}
	}
	ln, err := net.Listen("tcp", stubAddr)
	if err != nil {
		return fmt.Errorf("failed to open a socket for stub HTTP server: %w", err)
	}
	go startStubServer(ln)

	return nil
}

func ShutDown(ctx context.Context) error {
	var err1, err2, err3, err4 error
	if db != nil {
		err1 = db.Close()
		db = nil
	}
	if dbServer != nil {
		err2 = dbServer.Terminate(ctx)
		dbServer = nil
	}
	if mailServer != nil {
		err3 = mailServer.container.Terminate(ctx)
		mailServer = nil
	}
	if stubServer != nil {
		err4 = stubServer.Shutdown(ctx)
		stubServer = nil
	}
	return errors.Join(err1, err2, err3, err4)
}

// TODO: return an error if any
func TearDown() {
	// container.Restore force-kills open connections to the db, so a later
	// test could be handed a dead pooled connection and fail. Close/reopen
	// the pool around it so every test starts with a known-good connection.
	if err := db.Close(); err != nil {
		log.Printf("Failed to close DB before restore: %v\n", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := dbServer.Restore(ctx); err != nil {
		log.Printf("Failed to restore DB snapshot: %v\n", err)
	}
	if err := openDB(ctx); err != nil {
		log.Printf("Failed to reopen DB after restore: %v\n", err)
	}

	if mailServer != nil {
		if err := clearMailbox(ctx); err != nil {
			log.Printf("Failed to clear the mailbox: %v\n", err)
		}
	}

	clear(stubHTTPRules)
}

var dbServer *postgres.PostgresContainer

func startDBServer(ctx context.Context) error {
	var err error
	dbServer, err = postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithSQLDriver("pgx"),
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to launch test container: %w", err)
	}
	if err := openDB(ctx); err != nil {
		return fmt.Errorf("failed to open the DB for migration: %w", err)
	}
	if err := migration.Run(ctx, db, "up", nil); err != nil {
		return fmt.Errorf("failed to migrate DB: %w", err)
	}
	// container.Snapshot requires all DB connections to be closed.
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close DB before taking a snapshot: %w", err)
	}
	if err := dbServer.Snapshot(ctx); err != nil {
		return fmt.Errorf("failed to take a DB snapshot: %w", err)
	}
	if err := openDB(ctx); err != nil {
		return fmt.Errorf("failed to reopen the DB: %w", err)
	}
	return nil
}

type mailServerInstance struct {
	container *testcontainers.DockerContainer

	// The base URL for the Mailpit's REST API.
	apiURL string
}

var mailServer *mailServerInstance

// startMailServer launches a Mailpit server container and binds its SMTP listener to host:port.
func startMailServer(ctx context.Context, host netip.Addr, port string) error {
	if mailServer != nil {
		return fmt.Errorf("mail server is already running")
	}

	if !host.IsValid() {
		return errors.New("LaunchOption.SMTPServerHost is required to start the mail server")
	}
	if port == "" {
		return errors.New("LaunchOption.SMTPServerPort is required to start the mail server")
	}

	ctn, err := testcontainers.Run(ctx, "axllent/mailpit:latest",
		testcontainers.WithExposedPorts("1025/tcp", "8025/tcp"),
		testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
			hc.PortBindings = network.PortMap{
				network.MustParsePort("1025/tcp"): {{HostIP: host, HostPort: port}},
			}
		}),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/readyz").WithPort("8025/tcp").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to launch the mail server container: %w", err)
	}

	apiURL, err := ctn.PortEndpoint(ctx, "8025/tcp", "http")
	if err != nil {
		return fmt.Errorf("failed to resolve the mail server's API address: %w", err)
	}

	mailServer = &mailServerInstance{ctn, apiURL}
	return nil
}

// FindLastEmailTo returns the body of the most recent email the mail server has received for addr.
func FindLastEmailTo(ctx context.Context, addr string) (string, error) {
	if mailServer == nil {
		return "", errors.New("the mail server is not running")
	}

	// Search results are sorted by received date, newest first.
	query := url.QueryEscape(fmt.Sprintf(`to:"%s"`, addr))
	var found struct {
		Messages []struct{ ID string } `json:"messages"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/search?limit=1&query=%s", mailServer.apiURL, query), nil)
	if err != nil {
		return "", fmt.Errorf("failed to search the mailbox: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to search the mailbox: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("failed to search the mailbox: unexpected status %d: %s", res.StatusCode, body)
	}
	if err := json.NewDecoder(res.Body).Decode(&found); err != nil {
		return "", fmt.Errorf("failed to search the mailbox: %w", err)
	}
	if len(found.Messages) == 0 {
		return "", fmt.Errorf("no email found for %s", addr)
	}

	id := found.Messages[0].ID
	var email struct {
		HTML string
		Text string
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/message/%s", mailServer.apiURL, url.PathEscape(id)), nil)
	if err != nil {
		return "", fmt.Errorf("failed to read the email %s: %w", id, err)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to read the email %s: %w", id, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("failed to read the email %s: unexpected status %d: %s", id, res.StatusCode, body)
	}
	if err := json.NewDecoder(res.Body).Decode(&email); err != nil {
		return "", fmt.Errorf("failed to read the email %s: %w", id, err)
	}
	// Emails sent as text/html leave Text empty, and vice versa.
	if email.HTML != "" {
		return email.HTML, nil
	}
	return email.Text, nil
}

// clearMailbox deletes every email the mail server has stored.
func clearMailbox(ctx context.Context) error {
	// A DELETE with no message IDs in the body deletes them all.
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, mailServer.apiURL+"/api/v1/messages", nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, body)
	}
	return nil
}

var db *sql.DB

// DB returns the test database handle. It is only valid between [SetUp] and [ShutDown].
func DB() *sql.DB {
	return db
}

// openDB modifies the global db variable. Errors should be handled on the call site.
func openDB(ctx context.Context) error {
	dsn, err := dbServer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to construct the DSN: %w", err)
	}
	if db, err = sql.Open("pgx", dsn); err != nil {
		return fmt.Errorf("failed to connect to %s: %w", dsn, err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}
	return nil
}

var stubServer *http.Server

func startStubServer(ln net.Listener) {
	if stubServer != nil {
		panic("stub HTTP server is already running")
	}
	stubServer = &http.Server{
		Addr:    ln.Addr().String(),
		Handler: http.HandlerFunc(handleHTTPRequest),
	}
	if err := stubServer.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		panic(fmt.Sprintf("stub HTTP server exited abnormally: %v", err))
	}
}

var stubHTTPRules = map[string]string{}

// StubHTTP registers a rule so the stub server (which the API server uses as
// its outbound HTTP proxy) replies with the file at fp for requests to
// host+path. The content type is inferred from fp's extension:
//
//	testenv.StubHTTP("en.wikipedia.org", "/w/api.php", "./testdata/wikipedia_feed.xml")
//
// The stub only handles plain HTTP; it can't tunnel HTTPS, so stubbed requests
// must be made over http://.
func StubHTTP(host, path, fp string) {
	stubHTTPRules[host+path] = fp
}

func handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// Proxied requests carry an absolute URI, so the host is in r.URL.
	// Direct ones like "http://10.0.2.2:8081/clips/shared" leave it empty
	// and put the host in r.Host instead.
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	key := host + r.URL.Path
	fp, ok := stubHTTPRules[key]
	if !ok {
		http.Error(w, fmt.Sprintf("no stub rule found for %q", key), http.StatusNotFound)
		return
	}

	var mime string
	switch ext := filepath.Ext(fp); ext {
	case ".html":
		mime = "text/html; charset=utf-8"
	case ".json":
		mime = "application/json"
	case ".xml":
		mime = "application/xml; charset=utf-8"
	default:
		panic(fmt.Sprintf("unsupported stub file extension: %q", ext))
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		http.Error(w, fmt.Sprintf("request matched rule %q, but fixture not found in %q", key, fp), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", mime)
	if _, err := w.Write(data); err != nil {
		log.Printf("failed to write stub response for %q: %v", key, err)
	}
}
