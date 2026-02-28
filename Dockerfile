# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /wallet-server ./cmd/wallet-server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /outbox-consumer ./cmd/outbox-consumer

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /api /app/api
COPY --from=builder /wallet-server /app/wallet-server
COPY --from=builder /outbox-consumer /app/outbox-consumer
COPY db/migrations /app/db/migrations

EXPOSE 3100 4001

ENTRYPOINT ["/app/api"]
