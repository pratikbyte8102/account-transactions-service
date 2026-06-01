# Transactions Service

Small Go API for the transactions coding case.

Detailed architecture notes are available in [docs/architecture.MD](docs/architecture.MD).

## Requirements

- Go 1.26+
- PostgreSQL, or Docker with Docker Compose

## HTTP Stack

This project uses Go's built-in `net/http` package for all API routing, request handling, and server startup. No external HTTP framework is used.

The only external Go dependency is `github.com/lib/pq`, which is the PostgreSQL driver used by `database/sql`.

## Swagger UI

This project uses a specification-first Swagger UI setup with Go's standard library. It does not use Swaggo annotations, generated Go files, or the `swag init` command.

The OpenAPI 3.0 specification is maintained in:

```text
internal/http/swagger/openapi.yaml
```

The Swagger UI HTML page is maintained in:

```text
internal/http/swagger/index.html
```

Both files are embedded into the API binary with Go's `embed` package and served by the existing `net/http` router. No additional Go dependency is required.

After starting the application, open:

```text
http://localhost:8080/swagger/
```

Available documentation routes:

- `GET /swagger`: redirects to the Swagger UI page.
- `GET /swagger/`: serves the interactive Swagger UI page.
- `GET /swagger/openapi.yaml`: serves the OpenAPI specification.

The HTML page loads Swagger UI browser assets from `https://unpkg.com/swagger-ui-dist@5`, so the browser requires internet access to display the interactive interface. The OpenAPI specification itself is served locally by the API.

When adding or changing an endpoint, update `internal/http/swagger/openapi.yaml` and rebuild the application. For Docker:

```sh
docker compose up --build -d
```

## Run With Docker

```sh
./run.sh
```

The API will be available at `http://localhost:8080`.
Interactive Swagger UI documentation will be available at `http://localhost:8080/swagger/`.

Docker Compose starts two services:

- `api`: the Go backend, exposed on `localhost:8080`
- `postgres`: PostgreSQL 17, exposed on `localhost:5432`

The PostgreSQL data is stored in the Docker volume `account-transactions-service_postgres-data`, so data survives container restarts.

## Run Locally

Start PostgreSQL and set `DATABASE_URL` if it differs from the default:

```sh
DATABASE_URL='postgres://pratik@localhost:5432/transactions?sslmode=disable' \
PORT=8080 \
go run ./cmd/api
```

The application creates the required schema on startup.

## PostgreSQL Integration

The service connects to PostgreSQL using the `DATABASE_URL` environment variable.

Connection string format:

```sh
postgres://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=disable
```

Local development example:

```sh
export DATABASE_URL='postgres://pratik@localhost:5432/transactions?sslmode=disable'
go run ./cmd/api
```

Production example:

```sh
export DATABASE_URL='postgres://app_user:strong_password@prod-db.example.com:5432/transactions?sslmode=require'
export PORT=8080
go run ./cmd/api
```

Docker Compose uses this internal container URL:

```sh
DATABASE_URL='postgres://postgres:postgres@postgres:5432/transactions?sslmode=disable'
```

From your host machine, the same Docker database is reachable with:

```sh
postgres://postgres:postgres@localhost:5432/transactions?sslmode=disable
```

Inspect the Docker database with `psql` from inside the Postgres container:

```sh
docker compose exec postgres psql -U postgres -d transactions
```

Useful SQL commands:

```sql
\dt
SELECT * FROM accounts;
SELECT * FROM operation_types;
SELECT * FROM transactions;
```

Or run one query directly:

```sh
docker compose exec postgres psql -U postgres -d transactions \
  -c 'SELECT * FROM transactions ORDER BY transaction_id;'
```

Notes for production databases:

- Use the PostgreSQL URL provided by your hosting provider as `DATABASE_URL`.
- Prefer `sslmode=require` for managed production databases.
- URL-encode special characters in the username or password. For example, `@` becomes `%40`.
- The database user must be allowed to create tables because the app runs schema creation on startup.
- The app also reads `PORT`; if unset, it defaults to `8080`.

To verify a production connection before deployment:

```sh
DATABASE_URL='postgres://app_user:strong_password@prod-db.example.com:5432/transactions?sslmode=require' go test ./internal/repository -run TestPostgresRepositoryIntegration
```

## Test

```sh
go test ./...
```

## Clone And Run Checklist

For a new developer or reviewer:

```sh
git clone <repo-url>
cd account-transactions-service
chmod +x run.sh
./run.sh
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/health
```

Required tools:

- Docker Desktop, with the Docker engine running
- Docker Compose, included with current Docker Desktop versions

Go and PostgreSQL do not need to be installed on the host when using Docker.

## Endpoints

Swagger UI:

```text
http://127.0.0.1:8080/swagger/
```

Health check:

```sh
curl -i http://127.0.0.1:8080/health
```

Create an account:

```sh
curl -i -X POST http://127.0.0.1:8080/accounts \
  -H 'Content-Type: application/json' \
  -d '{"document_number":"12345678901"}'
```

Retrieve an account:

```sh
curl -i http://127.0.0.1:8080/accounts/1
```

Create a credit voucher transaction. Credit vouchers must be sent as positive amounts:

```sh
curl -i -X POST http://127.0.0.1:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"account_id":1,"operation_type_id":4,"amount":123.45}'
```

Create a normal purchase transaction. Operation types `1`, `2`, and `3` must be sent as negative amounts:

```sh
curl -i -X POST http://127.0.0.1:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"account_id":1,"operation_type_id":1,"amount":-50.00}'
```

Create a purchase with installments transaction:

```sh
curl -i -X POST http://127.0.0.1:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"account_id":1,"operation_type_id":2,"amount":-123.45}'
```

Create a withdrawal transaction:

```sh
curl -i -X POST http://127.0.0.1:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"account_id":1,"operation_type_id":3,"amount":-75.00}'
```

Invalid amount sign example:

```sh
curl -i -X POST http://127.0.0.1:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"account_id":1,"operation_type_id":1,"amount":123.45}'
```

Response:

```json
{"error":"amount must be negative for purchases and withdrawals"}
```

## Business Rules

- Operation type `1`: Normal Purchase, request amount must be negative.
- Operation type `2`: Purchase with installments, request amount must be negative.
- Operation type `3`: Withdrawal, request amount must be negative.
- Operation type `4`: Credit Voucher, request amount must be positive.
- Each transaction receives an `event_date` timestamp when it is created.
