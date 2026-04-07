# Interviewly

An AI-powered interview practice platform built as a monorepo:

- Backend API: Go (Gin, PostgreSQL, Redis)
- Frontend Web: Next.js (App Router, TypeScript, Tailwind)
- AI integration, voice interview, resume parsing, progress tracking, and subscription/payment features

This README is the main guide to run the full project locally or with Docker.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Repository Structure](#repository-structure)
- [Prerequisites](#prerequisites)
- [Quick Start (Docker Compose)](#quick-start-docker-compose)
- [Run Locally (Without Docker)](#run-locally-without-docker)
- [Environment Variables](#environment-variables)
- [Testing](#testing)
- [Deployment](#deployment)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

Main application flow:

1. Users sign in on the frontend (NextAuth with optional social providers).
2. The frontend calls backend APIs through proxy routes (`/api-proxy/*`) or direct base URL.
3. The backend handles authentication, job/resume parsing, interview sessions, feedback, progress, subscription, and payment.
4. Data is stored in PostgreSQL, while Redis is used for cache/rate limiting.
5. Optional third-party integrations: OpenAI-compatible API, ElevenLabs, Supabase S3/MinIO, Stripe.

## Repository Structure

```text
.
├── backend/                 # Go backend service
│   ├── cmd/server/          # Server entry point
│   ├── config/              # Environment-based configuration
│   ├── internal/            # Layered architecture (delivery/domain/repository/service/usecase)
│   ├── migrations/          # SQL migrations
│   └── scripts/             # Backend E2E scripts
├── frontend/                # Next.js frontend app
├── scripts/                 # Combined frontend-backend E2E scripts
├── docs/                    # Feature documentation
├── docker-compose.yml       # Local service orchestration
├── .env.example             # Example environment variables
└── Makefile                 # Common dev/test/migrate commands
```

## Prerequisites

Recommended minimum tools:

- Docker + Docker Compose
- Go 1.24+ (to run backend without Docker)
- Node.js 20+ and npm (to run frontend without Docker)
- Python 3 (for E2E scripts)

## Quick Start (Docker Compose)

1. Copy the environment file:

```bash
cp .env.example .env
```

2. Update at least these key variables in `.env`:

- `JWT_SECRET`
- `JWT_ISSUER`
- `DATABASE_URL`
- `REDIS_ADDR`

3. Build and start all services:

```bash
make compose-up
```

This command will:

- Start services defined in `docker-compose.yml`
- Run migrations (`make migrate-up`)

4. Check service status:

```bash
make compose-ps
```

5. Open the apps:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`

Important note:

- The PostgreSQL service in `docker-compose.yml` is currently commented out. Make sure `DATABASE_URL` points to an active database (for example Supabase/external Postgres), or enable local PostgreSQL service if needed.

Additional Docker commands:

```bash
make compose-logs           # tail logs for all services
make compose-down           # stop services
make compose-rebuild        # rebuild from scratch
make compose-reset          # reset images + volumes + migrate
make compose-clean-images   # remove compose images
```

## Run Locally (Without Docker)

### Backend

```bash
cd backend
go mod tidy
go run ./cmd/server
```

By default, backend listens on port `8080` (see `PORT`).

### Frontend

```bash
cd frontend
npm install
npm run dev
```

By default, frontend runs at `http://localhost:3000`.

For local development, make sure these values are set correctly:

- `NEXT_PUBLIC_API_BASE_URL` (example: `http://localhost:8080` or `/api-proxy`)
- `BACKEND_AUTH_BASE_URL` (example: `http://localhost:8080`)
- `NEXTAUTH_URL` (example: `http://localhost:3000`)
- `NEXTAUTH_SECRET`

## Environment Variables

The primary environment reference is `.env.example`.

Important variable groups:

1. Backend core
   - `PORT`, `APP_ENV`, `DATABASE_URL`
   - `POSTGRES_MAX_CONNS`, `POSTGRES_MIN_CONNS`, `POSTGRES_MAX_CONN_LIFETIME_MINUTES`
   - `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`

2. Auth
   - `JWT_SECRET`, `JWT_ISSUER`
   - `ACCESS_TOKEN_TTL_MINUTES`, `REFRESH_TOKEN_TTL_HOURS`

3. AI & Voice (optional)
   - `AI_PROVIDER`, `AI_MODEL`, `AI_API_BASE_URL`, `AI_API_KEY`
   - `VOICE_PROVIDER`, `ELEVENLABS_*`

4. Resume storage (optional)
   - `SUPABASE_*` or `MINIO_*`

5. Payment & Subscription (optional)
   - `STRIPE_*`
   - `VOICE_TOPUP_10_AMOUNT_IDR`, `VOICE_TOPUP_30_AMOUNT_IDR`

6. Frontend / OAuth
   - `NEXTAUTH_SECRET`, `NEXTAUTH_URL`
   - `BACKEND_AUTH_BASE_URL`, `NEXT_PUBLIC_API_BASE_URL`
   - `AUTH_GOOGLE_*`, `AUTH_AZURE_AD_*`

## Testing

### Backend unit/compile test

```bash
make test
```

### Backend E2E API regression

```bash
make test-e2e
```

### Frontend-backend E2E via Compose

```bash
python3 scripts/e2e_frontend_backend_compose.py
```

## Deployment

Backend deployment guide:

- `backend/DEPLOYMENT.md`

Detailed backend testing guide:

- `backend/TESTING.md`

## Troubleshooting

1. Backend cannot connect to DB
   - Check `DATABASE_URL`
   - Make sure DB host/port is reachable from backend container

2. Protected endpoints always return `401`
   - Check `JWT_SECRET` and `JWT_ISSUER`
   - Verify frontend/NextAuth login flow

3. AI features are not working
   - Check `AI_PROVIDER`, `AI_API_BASE_URL`, `AI_API_KEY`

4. Voice features are failing
   - Check `ELEVENLABS_API_KEY` and other `ELEVENLABS_*` variables

5. Migration fails
   - Run `make migrate-up` again
   - Verify database reachability and schema permissions

## Contribution Quick Guide

1. Create a feature branch.
2. Keep changes small and focused.
3. Run relevant tests (`make test`, `make test-e2e`).
4. Open a PR with a clear description and impact summary.
