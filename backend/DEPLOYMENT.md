# Backend Deployment (BE-603)

This backend can be deployed to Fly.io (recommended) or Railway using Docker.

## Required environment variables

- `PORT` (default: `8080`)
- `APP_ENV`
- `DATABASE_URL`
- `POSTGRES_MAX_CONNS`
- `POSTGRES_MIN_CONNS`
- `POSTGRES_MAX_CONN_LIFETIME_MINUTES`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `JWT_SECRET`
- `JWT_ISSUER`

## Run database migrations

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

Notes:

- Migration `000009_enforce_interview_reference_integrity` cleans orphan rows in `app_session_answers` and `app_feedback` before adding foreign key constraints.
- This ensures invalid `session_id` / `question_id` references are rejected at database level.

## Build and run locally with Docker

```bash
docker build -t interview-backend .
docker run --rm -p 8080:8080 --env PORT=8080 interview-backend
```

## Fly.io quick deploy

```bash
fly launch --copy-config --no-deploy
fly secrets set JWT_SECRET=your-secret DATABASE_URL=your-db-url
fly deploy
```

## Railway quick deploy

- Create new project from this repository.
- Set root directory to `backend`.
- Railway detects `Dockerfile` automatically.
- Configure environment variables from the list above.

## Regression test

Run backend API regression test before deploy:

```bash
make test-e2e
```
