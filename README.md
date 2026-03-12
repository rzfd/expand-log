# Expense Tracker Backend

Expense Tracker is a realistic Go backend portfolio project built with Echo, pgx, PostgreSQL, JWT auth, SQL migrations, and a simple background worker for recurring transactions.

## Stack

- Go
- Echo
- PostgreSQL
- pgx
- JWT
- bcrypt
- Docker Compose

## Project Structure

- `cmd/api`: API entrypoint
- `cmd/worker`: background worker entrypoint
- `internal/config`: environment-based configuration
- `internal/handler`: HTTP handlers and route registration
- `internal/middleware`: JWT middleware
- `internal/service`: business logic
- `internal/repository`: pgx-backed SQL repositories
- `internal/model`: domain models
- `internal/worker`: recurring transaction worker
- `internal/platform/postgres`: PostgreSQL connection bootstrap with retry
- `internal/pkg`: shared helpers
- `migrations`: SQL schema migrations

## Local Development

1. Copy the environment file:

```bash
cp .env.example .env
```

2. Start the stack:

```bash
docker compose up --build
```

This starts:

- `postgres`
- `migrate`
- `api`
- `worker`
- `pgadmin`

The API is available at `http://localhost:8080`.

PostgreSQL is exposed to the host on `${DB_HOST_PORT}` and the API/worker use the internal Compose address `postgres:5432`.
pgAdmin is available at `http://localhost:${PGADMIN_HOST_PORT}` with `${PGADMIN_DEFAULT_EMAIL}` / `${PGADMIN_DEFAULT_PASSWORD}`.

3. If you want to run migrations manually:

```bash
docker compose run --rm migrate
```

4. Stop the stack:

```bash
docker compose down
```

5. Reset the stack and remove PostgreSQL data:

```bash
docker compose down -v --remove-orphans
```

## Make Targets

```bash
make up
make down
make reset
make migrate-up
make test
make fmt
make vet
```

## API Routes

### Auth

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`

### Categories

- `GET /api/v1/categories`
- `POST /api/v1/categories`
- `PUT /api/v1/categories/:id`
- `DELETE /api/v1/categories/:id`

### Transactions

- `GET /api/v1/transactions`
- `GET /api/v1/transactions/:id`
- `POST /api/v1/transactions`
- `PUT /api/v1/transactions/:id`
- `DELETE /api/v1/transactions/:id`

### Reports

- `GET /api/v1/reports/monthly`
- `GET /api/v1/dashboard/summary`

### Budgets

- `GET /api/v1/budgets`
- `POST /api/v1/budgets`
- `PUT /api/v1/budgets/:id`
- `DELETE /api/v1/budgets/:id`

### Recurring Transactions

- `GET /api/v1/recurring-transactions`
- `POST /api/v1/recurring-transactions`
- `PUT /api/v1/recurring-transactions/:id`
- `DELETE /api/v1/recurring-transactions/:id`

## Request Notes

- Monetary fields use decimal strings, for example `"1250.50"`.
- Date fields use `YYYY-MM-DD`.
- JWT is issued by `POST /api/v1/auth/login` and passed as `Authorization: Bearer <token>`.

## Postman

Import `postman/expense-tracker.postman_collection.json` into Postman for a ready-to-run local collection. The `Login` request stores the JWT automatically, and the create requests store returned IDs in collection variables.

## Sample cURL

Register:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

`register` creates the user account only. It does not return a JWT.

Login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

Create category:

```bash
curl -X POST http://localhost:8080/api/v1/categories \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Food","type":"expense"}'
```

Create transaction:

```bash
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"category_id":1,"type":"expense","amount":"49.90","note":"Lunch","transaction_date":"2026-03-12"}'
```

Filter transactions:

```bash
curl "http://localhost:8080/api/v1/transactions?start_date=2026-03-01&end_date=2026-03-31&type=expense&page=1&page_size=20" \
  -H "Authorization: Bearer <TOKEN>"
```

Get monthly report:

```bash
curl "http://localhost:8080/api/v1/reports/monthly?year=2026&month=3" \
  -H "Authorization: Bearer <TOKEN>"
```

Create budget:

```bash
curl -X POST http://localhost:8080/api/v1/budgets \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"category_id":1,"year":2026,"month":3,"amount":"500.00"}'
```

Create recurring transaction:

```bash
curl -X POST http://localhost:8080/api/v1/recurring-transactions \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"category_id":1,"type":"expense","amount":"19.99","note":"Subscription","frequency":"monthly","start_date":"2026-03-01","active":true}'
```

## Recurring Worker Behavior

- The worker polls the database on the configured interval.
- A recurring rule stores `next_run_date`.
- When due, the worker creates a transaction with source `recurring`.
- Duplicate generation is prevented with a unique database index on `(recurring_transaction_id, transaction_date)`.

## Current Scope

- Local Docker Compose workflow only
- No Redis
- No observability stack
- No CI/CD or cloud manifests
- No external reverse proxy
