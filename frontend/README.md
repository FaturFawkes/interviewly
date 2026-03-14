# AI Interview Coach Frontend

Premium dark-mode AI SaaS frontend for interview practice, including:

- High-conversion landing page
- Dashboard with score analytics
- Job description + resume analysis workflow
- Live interview practice flow with AI feedback
- Progress analytics page

Built with Next.js App Router, TypeScript, Tailwind CSS, and reusable modular components.

## Tech Stack

- Next.js 16 (App Router)
- React 19
- TypeScript
- Tailwind CSS v4
- Recharts (analytics visualizations)
- Lucide React (icons)

## Run Locally

```bash
npm install
npm run dev
```

Set the backend URL:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

## Docker Compose (Frontend + Backend)

From repository root:

```bash
docker compose up --build -d
docker compose --profile tools run --rm migrate
python3 scripts/e2e_frontend_backend_compose.py
```

This validates:

- Frontend route availability
- Frontend proxy to backend (`/api-proxy/*`)
- Authenticated API flow (`/api/me`, questions, session, answer, feedback, progress)

## App Routes

- `/` → Landing page
- `/dashboard` → Score overview, sessions, recommendations
- `/upload` → Resume + JD parsing and skills extraction
- `/interview` → AI question practice, answer submission, feedback
- `/progress` → Long-term analytics and improvement guidance

## API Integration

Frontend uses existing backend routes:

- `POST /api/job/parse`
- `POST /api/resume`
- `POST /api/questions/generate`
- `POST /api/session/start`
- `POST /api/session/answer`
- `POST /api/feedback/generate`
- `GET /api/progress`
- `GET /api/session/history`

API client files:

- `lib/api/client.ts`
- `lib/api/endpoints.ts`
- `lib/api/types.ts`

## Design System

The UI token system is defined in:

- `app/globals.css` (CSS vars + utility surfaces)
- `lib/design-system.ts` (typed token map)

Key visual direction:

- Dark base (`#0B0F14`)
- Purple-to-blue gradients (`#7B61FF → #2F80ED`)
- Cyan accent glow (`#00E5FF`)
- Glassmorphism cards and soft gradient orbs

## Notes on Authentication

The app now uses real social login flow:

- Frontend OAuth with NextAuth (`Google` and `Microsoft Entra ID`)
- Backend exchange endpoint `POST /auth/social-login`
- Automatic user registration on first social login (upsert by email)
- Protected app routes (`/dashboard`, `/upload`, `/interview`, `/progress`) require login

Required frontend environment variables:

```bash
NEXTAUTH_SECRET=your-random-secret
NEXTAUTH_URL=http://localhost:3000
BACKEND_AUTH_BASE_URL=http://localhost:8080

AUTH_GOOGLE_ID=...
AUTH_GOOGLE_SECRET=...

AUTH_AZURE_AD_ID=...
AUTH_AZURE_AD_SECRET=...
AUTH_AZURE_AD_TENANT_ID=<tenant-id>
```

Token bridge utilities:

- `lib/auth/token-provider.ts`
