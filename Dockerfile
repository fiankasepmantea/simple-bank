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

# Install migrate
RUN apk add --no-cache curl tar \
 && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz \
 | tar -xz \
 && chmod +x migrate

# =========================
# Run stage
# =========================
FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY app.env .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./migration

RUN chmod +x /app/start.sh /app/migrate /app/wait-for.sh

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
