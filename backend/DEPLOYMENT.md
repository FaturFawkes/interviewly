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
- `SUPABASE_URL` (opsional; dipakai untuk derivasi endpoint jika `SUPABASE_S3_ENDPOINT` kosong)
- `SUPABASE_S3_ENDPOINT` (opsional; format resmi: `https://<project-ref>.storage.supabase.co/storage/v1/s3`)
- `SUPABASE_S3_REGION` (default: `us-east-1`, gunakan region dari Storage settings)
- `SUPABASE_S3_ACCESS_KEY_ID` (opsional, wajib jika ingin upload file resume ke Supabase S3)
- `SUPABASE_S3_SECRET_ACCESS_KEY` (opsional, wajib jika ingin upload file resume ke Supabase S3)
- `SUPABASE_STORAGE_BUCKET` (default: `resumes`)
- `SUPABASE_STORAGE_PATH_PREFIX` (opsional, mis. `prod`)

## Stripe environment variables (if payments enabled)

- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `STRIPE_SUCCESS_URL`
- `STRIPE_CANCEL_URL`
- `STRIPE_CURRENCY` (default: `idr`)
- `STRIPE_PRICE_STARTER_MONTHLY`
- `STRIPE_PRICE_PRO_MONTHLY`
- `STRIPE_PRICE_ELITE_MONTHLY`
- `STRIPE_PRICE_VOICE_TOPUP_10`
- `STRIPE_PRICE_VOICE_TOPUP_30`
- `VOICE_TOPUP_10_AMOUNT_IDR` (fallback nominal jika `STRIPE_PRICE_VOICE_TOPUP_10` kosong)
- `VOICE_TOPUP_30_AMOUNT_IDR` (fallback nominal jika `STRIPE_PRICE_VOICE_TOPUP_30` kosong)

Notes:

- Voice add-on tidak wajib memiliki Product/Price statis di Stripe. Jika `STRIPE_PRICE_VOICE_TOPUP_*` kosong, backend akan membuat `price_data` dinamis saat checkout.
- Jika ingin kontrol katalog/harga dari Stripe Dashboard, buat Product+Price di Stripe lalu isi `STRIPE_PRICE_VOICE_TOPUP_10/30`.

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
