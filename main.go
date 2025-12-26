package main

import (
	"context"
	"log"

	"simple-bank/api"
	db "simple-bank/db/sqlc"
	"simple-bank/db/util"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Create connection pool
	pool, err := pgxpool.New(context.Background(), config.DBSOURCE)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer pool.Close()

	// Initialize store & server
	store := db.NewStore(pool) // Mengembalikan Store (interface)
	server := api.NewServer(store) // Terima Store (interface)

	// Start HTTP server
	log.Printf("Starting server on %s", config.SERVER_ADDRESS)
	if err := server.Start(config.SERVER_ADDRESS); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
