# =========================
# Build stage
# =========================
FROM golang:1.25.5-alpine3.23 AS builder

WORKDIR /app

COPY . .

RUN go build -o main main.go

RUN apk add --no-cache curl tar

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz \
    | tar -xz

# =========================
# Run stage
# =========================
FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate

COPY app.env .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./migration

RUN chmod +x /app/start.sh /app/migrate

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
