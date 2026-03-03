package gapi

import (
	"context"
	db "simple-bank/db/sqlc"
	"simple-bank/pb"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	if err := validateCurrency(req.GetCurrency()); err != nil {
		return nil, invalidArgumentError([]*errdetails.BadRequest_FieldViolation{
			fieldViolation("currency", err),
		})
	}

	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.GetCurrency(),
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case "23503": // foreign_key_violation
				return nil, status.Errorf(codes.FailedPrecondition, "owner doesn't exist")
			case "23505": // unique_violation
				return nil, status.Errorf(codes.AlreadyExists, "account already exists for this currency")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %s", err)
	}

	return &pb.CreateAccountResponse{
		Account: convertAccount(account),
	}, nil
}

func validateCurrency(currency string) error {
	supported := map[string]bool{"USD": true, "EUR": true, "CAD": true}
	if !supported[currency] {
		return status.Errorf(codes.InvalidArgument, "currency %s is not supported", currency)
	}
	return nil
}