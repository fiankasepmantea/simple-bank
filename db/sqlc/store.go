package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store adalah interface yang mencakup semua operasi yang dibutuhkan
type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
}
// SQLStore mengimplementasi Store
type SQLStore struct {
	db *pgxpool.Pool
	*Queries
}
// NewStore mengembalikan interface Store
func NewStore(db *pgxpool.Pool) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// execTx menjalankan fungsi dalam transaksi
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return err
	}

	q := store.WithTx(tx)

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = fn(q)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}