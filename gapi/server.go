package gapi

import (
	"fmt"
	db "simple-bank/db/sqlc"
	"simple-bank/db/util"
	"simple-bank/pb"
	"simple-bank/token"
	"simple-bank/worker"
	"github.com/gin-gonic/gin"
)

// Server serve gRPC requests for our banking service
type Server struct {
	pb.UnimplementedSimpleBankServer
	config util.Config
	store  db.Store
	tokenMaker token.Maker	
	router *gin.Engine
	taskDistributor worker.TaskDistributor
}

// New Server craetes a new gRPC server
func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config: config,
		store: store,
		tokenMaker: tokenMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}