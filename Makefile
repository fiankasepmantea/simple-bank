# =============================
# Configuration
# =============================

DB_SOURCE ?= postgresql://postgres:root@localhost:54322/simple_bank?sslmode=disable
#DB_SOURCE ?= postgresql://postgresprod:rootprod@simple-bank.czmqqew0g1mp.ap-southeast-3.rds.amazonaws.com:5432/simple_bank

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

db_docs:
	dbdocs build doc/db.dbml

db_schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

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

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
# 	go install github.com/rakyll/statik@latest

proto:
	rm -f pb/*.go
	rm -f doc/swagger/*.swagger.json
	protoc --proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
		proto/*.proto
# 	statik -src=./doc/swagger -dest=./doc

evans:
	evans --host localhost --port 9090 -r repl

redis:
	docker run --name redis-simple-bank -p 6380:6379 -d redis:8-alpine

# =============================
# Phony Targets
# =============================
.PHONY: postgres createdb dropdb migrateup migratedown db_docs sqlc db_schema mock test server install-tools proto evans redis
