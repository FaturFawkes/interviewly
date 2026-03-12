# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/server ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /bin/server /app/server

EXPOSE 8080
USER app

ENV PORT=8080
CMD ["/app/server"]
