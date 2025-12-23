package main

import (
	"context"
	"log"

	"simplebankfian/api"
	db "simplebankfian/db/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbSource      = "postgresql://postgres:root@localhost:54322/simple_bank?sslmode=disable"
	serverAddress = "0.0.0.0:8080"
)

func main() {
	// Create connection pool
	pool, err := pgxpool.New(context.Background(), dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer pool.Close()

	// Initialize store & server
	store := db.NewStore(pool)
	server := api.NewServer(store)

	// Start HTTP server
	log.Printf("Starting server on %s", serverAddress)
	if err := server.Start(serverAddress); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
