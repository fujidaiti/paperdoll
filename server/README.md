# Server

## Development Infrastructure

The development infrastructure is defined in `docker-compose.yml`. Run this
command to start all services:

```sh
docker compose up -d
```

Create `server/.env` from the example. The example values are configured for the
services above:

```sh
cp server/.env.example server/.env
```

Apply the database migrations:

```sh
make db-migrate
```

### Mailpit

Sent emails are visible at http://localhost:8025.

### Postgres

To open a SQL shell:

```sh
docker compose exec postgres psql -U app -d app
```

Data is stored in a named volume and is kept across `docker compose down`. To
delete all data, run `docker compose down -v`.
