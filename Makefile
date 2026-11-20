-include server/.env
export

# Set up dev environment. Run this command once after cloning.
.PHONY: setup
setup:
	git config core.hooksPath .githooks
	fvm use
	fvm flutter pub get
	go mod tidy
	npm ci

.PHONY: db-migrate
db-migrate:
	go run ./server/cmd/app/main.go migrate up

.PHONY: dev-server
dev-server:
	go run ./server/cmd/app/main.go serve

.PHONY: build-server-linux-amd64
build-server-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o build/paperdoll-server-linux-amd64 ./server/cmd/app/main.go
