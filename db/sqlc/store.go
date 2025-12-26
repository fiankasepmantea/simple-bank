		package db

		import (
			"context"
			"github.com/jackc/pgx/v5/pgxpool"
			"github.com/jackc/pgx/v5"   
		)

		// Store adalah interface yang mencakup semua operasi yang dibutuhkan
		type Store interface {
			Querier
			TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
		}
		// TransferTxParams contains the input parameters of the transfer transaction
		type TransferTxParams struct {
			FromAccountID int64
			ToAccountID   int64
			Amount        int64
		}

		// TransferTxResult is the result of the transfer transaction
		type TransferTxResult struct {
			Transfer    Transfer
			FromAccount Account
			ToAccount   Account
			FromEntry   Entry
			ToEntry     Entry
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

		// WithTx mengembalikan Queries baru dengan transaksi
		func (store *SQLStore) WithTx(tx pgx.Tx) *Queries {
			return &Queries{db: tx}
		}

		func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
			var result TransferTxResult

			err := store.execTx(ctx, func(q *Queries) error {
				var err error

				// Create transfer record
				result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
					FromAccountID: arg.FromAccountID,
					ToAccountID:   arg.ToAccountID,
					Amount:        arg.Amount,
				})
				if err != nil {
					return err
				}

				// Create entries
				result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
					AccountID: arg.FromAccountID,
					Amount:    -arg.Amount,
				})
				if err != nil {
					return err
				}

				result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
					AccountID: arg.ToAccountID,
					Amount:    arg.Amount,
				})
				if err != nil {
					return err
				}

				// Update account balances
				if arg.FromAccountID < arg.ToAccountID {
					result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
				} else {
					result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
				}
				return err
			})

			return result, err
		}

		func addMoney(
			ctx context.Context,
			q *Queries,
			accountID1 int64,
			amount1 int64,
			accountID2 int64,
			amount2 int64,
		) (Account, Account, error) {
			account1, err := q.AddAccountBalance(ctx, AddAccountBalanceParams{
				ID:     accountID1,
				Amount: amount1,
			})
			if err != nil {
				return Account{}, Account{}, err
			}

			account2, err := q.AddAccountBalance(ctx, AddAccountBalanceParams{
				ID:     accountID2,
				Amount: amount2,
			})
			if err != nil {
				return Account{}, Account{}, err
			}

			return account1, account2, nil
		}