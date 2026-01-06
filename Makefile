# =============================
# Configuration
# =============================

#DB_SOURCE ?= postgresql://postgres:root@localhost:54322/simple_bank?sslmode=disable
DB_SOURCE ?= postgresql://postgresprod:rootprod@simple-bank.czmqqew0g1mp.ap-southeast-3.rds.amazonaws.com:5432/simple_bank

# Tool versions
MOCKGEN_VERSION ?= v1.6.0

# =============================
# Docker / Database
# =============================
postgres:
	docker run --name postgres14 --network bank-network -p 54322:5432 \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=root \
		-d postgres:alpine3.22

createdb:
	docker exec postgres14 psql -U postgres -c "CREATE DATABASE simple_bank;"

dropdb:
	docker exec postgres14 psql -U postgres -c "DROP DATABASE IF EXISTS simple_bank;"

# =============================
# Migrations
# =============================
migrateup:
	migrate -path $(PWD)/db/migration -database $(DB_SOURCE) -verbose up

migrateup1:
	migrate -path $(PWD)/db/migration -database $(DB_SOURCE) -verbose up 1

migratedown:
	migrate -path $(PWD)/db/migration -database $(DB_SOURCE) -verbose down

migratedown1:
	migrate -path $(PWD)/db/migration -database $(DB_SOURCE) -verbose down 1

# =============================
# Code Generation
# =============================
sqlc:
	sqlc generate

# Generate GoMock mocks (Store interface)
mock:
	GOFLAGS=-mod=mod go run github.com/golang/mock/mockgen@v1.6.0 \
		-package=mockdb \
		-destination=db/mock/store.go \
		simple-bank/db/sqlc Store

# =============================
# Testing / Run
# =============================
test:
	go test -v -cover ./...

server:
	go run main.go

# =============================
# Phony Targets
# =============================
.PHONY: postgres createdb dropdb migrateup migratedown sqlc mock test server
