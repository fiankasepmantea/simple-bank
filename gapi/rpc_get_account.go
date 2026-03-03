package gapi

import (
	"context"
	// db "simple-bank/db/sqlc"
	"simple-bank/pb"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	account, err := server.store.GetAccount(ctx, req.GetId())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get account: %s", err)
	}

	if account.Owner != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "account doesn't belong to authenticated user")
	}

	return &pb.GetAccountResponse{
		Account: convertAccount(account),
	}, nil
}