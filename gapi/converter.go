package gapi

import (
	db "simple-bank/db/sqlc"
	"simple-bank/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertUser(user db.User) *pb.User {
	var passwordChangedAt *timestamppb.Timestamp
	if user.PasswordChangedAt.Valid {
		passwordChangedAt = timestamppb.New(user.PasswordChangedAt.Time)
	}

	var createdAt *timestamppb.Timestamp
	if user.CreatedAt.Valid {
		createdAt = timestamppb.New(user.CreatedAt.Time)
	}

	return &pb.User{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: passwordChangedAt,
		CreatedAt:         createdAt,
	}
}
func convertAccount(account db.Account) *pb.Account {
	return &pb.Account{
		Id:        account.ID,
		Owner:     account.Owner,
		Balance:   account.Balance,
		Currency:  account.Currency,
		CreatedAt: timestamppb.New(account.CreatedAt.Time),
	}
}

func convertTransfer(transfer db.Transfer) *pb.Transfer {
	return &pb.Transfer{
		Id:            transfer.ID,
		FromAccountId: transfer.FromAccountID,
		ToAccountId:   transfer.ToAccountID,
		Amount:        transfer.Amount,
		CreatedAt:     timestamppb.New(transfer.CreatedAt.Time),
	}
}

func convertEntry(entry db.Entry) *pb.Entry {
	return &pb.Entry{
		Id:         entry.ID,
		AccountId:  entry.AccountID,
		Amount:     entry.Amount,
		CreatedAt:  timestamppb.New(entry.CreatedAt.Time),
	}
}