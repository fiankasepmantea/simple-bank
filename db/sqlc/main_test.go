package db

import (
	"context"
	"log"
	"os"
	"simple-bank/db/util"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testQueries *Queries
var testDB *pgxpool.Pool
var testStore Store
func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDB, err = pgxpool.New(context.Background(), config.DBSOURCE)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB)
	testStore = NewStore(testDB)

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}