package gapi

import (
	"context"
	db "simple-bank/db/sqlc"
	"simple-bank/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	arg := db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.GetPageSize(),
		Offset: (req.GetPageId() - 1) * req.GetPageSize(),
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %s", err)
	}

	resp := &pb.ListAccountsResponse{}
	for _, acc := range accounts {
		resp.Accounts = append(resp.Accounts, convertAccount(acc))
	}

	return resp, nil
}