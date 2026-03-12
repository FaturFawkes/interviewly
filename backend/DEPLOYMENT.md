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
