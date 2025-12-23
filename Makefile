postgres:
	docker run --name postgres14 -p 54322:5432 \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=root \
		-d postgres:alpine3.22

createdb:
	docker exec postgres14 psql -U postgres -c "CREATE DATABASE simple_bank;"

dropdb:
	docker exec postgres14 psql -U postgres -c "DROP DATABASE IF EXISTS simple_bank;"

migrateup:
	migrate -path $(PWD)/db/migration \
		-database "postgresql://postgres:root@localhost:54322/simple_bank?sslmode=disable" \
		-verbose up

migratedown:
	migrate -path $(PWD)/db/migration \
		-database "postgresql://postgres:root@localhost:54322/simple_bank?sslmode=disable" \
		-verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server