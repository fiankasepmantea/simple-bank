package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbSource = "postgresql://postgres:root@localhost:54322/simple_bank?sslmode=disable"
)

var testQueries *Queries
var testDB *pgxpool.Pool
var testStore *Store
func TestMain(m *testing.M) {
	var err error
	testDB, err = pgxpool.New(context.Background(), dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB)
	testStore = NewStore(testDB)

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}