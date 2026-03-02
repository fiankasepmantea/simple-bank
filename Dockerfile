# =========================
# Build stage
# =========================
FROM golang:1.25.5-alpine3.23 AS builder

WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN go build -o main main.go

# =========================
# Run stage
# =========================
FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/main .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./db/migration

RUN chmod +x /app/start.sh /app/wait-for.sh

EXPOSE 8080 9090

ENTRYPOINT ["/app/start.sh"]
