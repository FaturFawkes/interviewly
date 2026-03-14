# syntax=docker/dockerfile:1

FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ARG NEXT_PUBLIC_API_BASE_URL=/api-proxy
ARG NEXT_PUBLIC_DEV_JWT_TOKEN=
ARG BACKEND_INTERNAL_URL=http://backend:8080
ENV NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL}
ENV NEXT_PUBLIC_DEV_JWT_TOKEN=${NEXT_PUBLIC_DEV_JWT_TOKEN}
ENV BACKEND_INTERNAL_URL=${BACKEND_INTERNAL_URL}
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/next.config.ts ./next.config.ts
EXPOSE 3000
CMD ["npm", "run", "start"]
