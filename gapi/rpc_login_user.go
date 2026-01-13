package gapi

import (
	"context"
	"errors"

	db "simple-bank/db/sqlc"
	"simple-bank/db/util"
	"simple-bank/pb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	user, err := server.store.GetUser(ctx, req.GetUsername())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to find user")
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "incorrect password")
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.AccessTokenDuration,
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create refresh token")
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create refresh token")
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID: pgtype.UUID{
			Bytes: refreshPayload.ID,
			Valid: true,
		},
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    "",
		ClientIp:     "",
		IsBlocked:    false,
		ExpiresAt: pgtype.Timestamptz{
			Time:  refreshPayload.ExpiredAt,
			Valid: true,
		},
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session")
	}

	rsp := &pb.LoginUserResponse{
		User: convertUser(user),
		SessionId: session.ID.String(),
		AccessToken: accessToken,
		RefreshToken: refreshToken,
		AccessTokenExpiresAt:    timestamppb.New(accessPayload.ExpiredAt),
		RefreshTokenExpiresAt:   timestamppb.New(refreshPayload.ExpiredAt),
	}

	return rsp, nil
}
