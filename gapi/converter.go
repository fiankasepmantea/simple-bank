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