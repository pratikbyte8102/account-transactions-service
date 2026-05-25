# Architecture

## Overview

This service is a small layered Go API for the transactions coding case. It exposes HTTP endpoints with Go's standard `net/http` package, applies business rules in a service layer, and stores data in PostgreSQL through `database/sql`.

The design keeps framework and infrastructure details at the edges. HTTP handlers do not contain transaction business rules, and the service layer does not know SQL details.

## Architecture Diagram

```text
Client / curl
    |
    v
net/http server
    |
    v
HTTP handlers
    |
    v
Service layer
    |
    v
Repository interfaces
    |
    v
PostgreSQL repositories
    |
    v
PostgreSQL database
```

Docker local flow:

```text
Host machine
    |
    | curl http://127.0.0.1:8080
    v
Docker service: api
    |
    | DATABASE_URL=postgres://postgres:postgres@postgres:5432/transactions?sslmode=disable
    v
Docker service: postgres
    |
    v
Docker volume: postgres-data
```

## Folder Structure

```text
cmd/api
```

Application entrypoint. It loads configuration, opens the database connection, runs schema setup, wires dependencies, and starts the HTTP server.

```text
internal/config
```

Environment-based configuration. The app reads `PORT` and `DATABASE_URL`.

```text
internal/domain
```

Core domain types and rules that do not depend on HTTP or PostgreSQL. This includes account and transaction models, operation type constants, and amount normalization.

```text
internal/service
```

Business use cases. This layer validates input, checks account existence, normalizes transaction amounts, assigns event timestamps, and calls repository interfaces.

```text
internal/http
```

HTTP transport layer. It owns routes, request DTOs, response DTOs, JSON decoding, status codes, and error responses.

```text
internal/repository
```

PostgreSQL persistence. This package implements the repository interfaces required by the service layer using `database/sql` and `github.com/lib/pq`.

```text
migrations
```

SQL schema reference for the database tables and seed data. The same schema is also executed on startup from the repository package so Docker/local startup is simple.

```text
docs
```

Project documentation beyond the quick-start README.

## Request Flow

### Create Account

```text
POST /accounts
    -> HTTP handler decodes {"document_number": "..."}
    -> service validates document_number
    -> account repository inserts into accounts
    -> HTTP handler returns account_id and document_number
```

### Get Account

```text
GET /accounts/{accountId}
    -> HTTP handler parses accountId from path
    -> service validates account_id
    -> account repository reads from accounts
    -> HTTP handler returns account data or 404
```

### Create Transaction

```text
POST /transactions
    -> HTTP handler decodes account_id, operation_type_id, amount
    -> service validates account, operation type, and non-zero amount
    -> service normalizes amount sign
    -> transaction repository inserts into transactions
    -> HTTP handler returns the stored transaction
```

Amount rules:

- Operation type `1`, Normal Purchase: stored negative.
- Operation type `2`, Purchase with installments: stored negative.
- Operation type `3`, Withdrawal: stored negative.
- Operation type `4`, Credit Voucher: stored positive.

## Database

Database name:

```text
transactions
```

Docker Compose defines it with:

```yaml
POSTGRES_DB: transactions
```

Main tables:

- `accounts`: account id and cardholder document number.
- `operation_types`: seed data for operation types `1` through `4`.
- `transactions`: account id, operation type id, signed amount, and event date.

Inspect data in Docker:

```sh
docker compose exec postgres psql -U postgres -d transactions
```

Useful SQL:

```sql
\dt
SELECT * FROM accounts;
SELECT * FROM operation_types;
SELECT * FROM transactions;
```

## Testing Layout

The tests intentionally live beside the code they test:

```text
internal/http/handler_test.go
internal/service/service_test.go
internal/repository/postgres_test.go
```

This is standard Go convention. Unlike Spring Boot projects, Go does not normally use a separate `src/test` module. A file ending in `_test.go` is compiled only for tests and is ignored by normal application builds.

The test strategy is:

- Service unit tests cover business rules and validation.
- HTTP handler tests cover JSON contracts and status codes without a live database.
- Repository integration test is skipped by default and runs only when `TEST_DATABASE_URL` is set.

Run all default tests:

```sh
go test ./...
```

Run the optional PostgreSQL repository integration test:

```sh
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/transactions?sslmode=disable' \
go test ./internal/repository -run TestPostgresRepositoryIntegration
```

## Design Decisions

- `net/http` is used instead of an external HTTP framework to keep the service simple and idiomatic.
- Business rules are centralized in the service layer, not duplicated in handlers or repositories.
- Repository interfaces are defined where they are consumed, in the service package.
- PostgreSQL implementation is behind those interfaces, so tests can use fakes.
- Docker Compose is the preferred local setup for reviewers because it avoids requiring local Go or PostgreSQL installation.
- Schema setup runs on startup for the coding exercise. For a larger production system, this could be replaced by an external migration runner.
