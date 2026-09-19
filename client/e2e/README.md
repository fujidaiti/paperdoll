# E2E tests

The real-backend E2E suite, run with Patrol on a real device/emulator. Unlike
the widget tests in `client/test/`, these drive the actual Flutter app against a
real API server and DB (see the doc comment at the top of `main.go` for how the
runner, client, and test DB fit together).

## Requirements

Before running, check that:

- An Android emulator is running (`emulator-5554` by default):
  `fvm flutter devices`
- Docker is running (the test DB and a mail server, which receives the emails
  the API server sends, are spun up in containers via testcontainers):
  `docker info > /dev/null 2>&1 && echo "docker is running"`
- Nothing else is bound to ports 1026 and 8026, which the mail server binds to.
  These are one above the ports `docker-compose.yml` uses, so the development
  mail server can stay up.

## Running

```sh
cd e2e && go run ./...
```

## Running specific test cases

Everything after a literal `--` is forwarded to the underlying `patrol test`
invocation, so you can use Patrol's own `--tags` and `--target` options to
narrow down which tests run instead of executing the whole suite. Each test has
a short, unique tag (e.g. `signup`, `signin`, `signout`, `today-read`):

```sh
go run ./... -- --tags signin
go run ./... -- --target e2e/auth_test.dart
go run ./... -- --target e2e/auth_test.dart --tags signin
```
