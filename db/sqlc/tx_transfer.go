package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"encoding/json"
)

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
// WithTx mengembalikan Queries baru dengan transaksi
func (store *SQLStore) WithTx(tx pgx.Tx) *Queries {
	return &Queries{db: tx}
}

func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// 1️⃣ Create transfer record
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		// 2️⃣ Create debit entry
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		// 3️⃣ Create credit entry
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// 4️⃣ Update account balances (deadlock-safe ordering)
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(
				ctx,
				q,
				arg.FromAccountID, -arg.Amount,
				arg.ToAccountID, arg.Amount,
			)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(
				ctx,
				q,
				arg.ToAccountID, arg.Amount,
				arg.FromAccountID, -arg.Amount,
			)
		}
		if err != nil {
			return err
		}

		// 5️⃣ INSERT OUTBOX EVENT (CRITICAL PART 🔥)

		eventPayload := map[string]interface{}{
			"transfer_id":  result.Transfer.ID,
			"from_account": arg.FromAccountID,
			"to_account":   arg.ToAccountID,
			"amount":       arg.Amount,
		}

		payloadBytes, err := json.Marshal(eventPayload)
		if err != nil {
			return err
		}

		_, err = q.CreateOutboxEvent(ctx, CreateOutboxEventParams{
			AggregateType: "transfer",
			AggregateID:   result.Transfer.ID,
			EventType:     "transfer_created",
			Payload:       payloadBytes,
		})
		if err != nil {
			return err
		}

		return nil
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


