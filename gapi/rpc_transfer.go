package gapi

import (
	"context"
	"fmt"
	db "simple-bank/db/sqlc"
	"simple-bank/pb"

	"github.com/jackc/pgx/v5"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	if err := validateTransferRequest(req); err != nil {
		return nil, invalidArgumentError(err)
	}

	// Validate from_account ownership
	fromAccount, err := server.store.GetAccount(ctx, req.GetFromAccountId())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "from_account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get from_account: %s", err)
	}
	if fromAccount.Owner != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "from_account doesn't belong to authenticated user")
	}

	// Validate currency match
	if fromAccount.Currency != req.GetCurrency() {
		return nil, status.Errorf(codes.InvalidArgument, "from_account currency mismatch")
	}

	// Validate to_account
	toAccount, err := server.store.GetAccount(ctx, req.GetToAccountId())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "to_account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get to_account: %s", err)
	}
	if toAccount.Currency != req.GetCurrency() {
		return nil, status.Errorf(codes.InvalidArgument, "to_account currency mismatch")
	}

	// Execute transfer
	arg := db.TransferTxParams{
		FromAccountID: req.GetFromAccountId(),
		ToAccountID:   req.GetToAccountId(),
		Amount:        req.GetAmount(),
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to transfer: %s", err)
	}

	return &pb.TransferResponse{
		Transfer:    convertTransfer(result.Transfer),
		FromAccount: convertAccount(result.FromAccount),
		ToAccount:   convertAccount(result.ToAccount),
		FromEntry:   convertEntry(result.FromEntry),
		ToEntry:     convertEntry(result.ToEntry),
	}, nil
}

func validateTransferRequest(req *pb.TransferRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if req.GetFromAccountId() <= 0 {
		violations = append(violations, fieldViolation("from_account_id", fmt.Errorf("must be positive")))
	}
	if req.GetToAccountId() <= 0 {
		violations = append(violations, fieldViolation("to_account_id", fmt.Errorf("must be positive")))
	}
	if req.GetAmount() <= 0 {
		violations = append(violations, fieldViolation("amount", fmt.Errorf("must be positive")))
	}
	if err := validateCurrency(req.GetCurrency()); err != nil {
		violations = append(violations, fieldViolation("currency", err))
	}
	return violations
}