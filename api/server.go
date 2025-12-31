package api

import (
	"fmt"
	db "simple-bank/db/sqlc"
	"simple-bank/db/util"
	"simple-bank/token"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Server serve HTTP requests for our banking service
type Server struct {
	config util.Config
	store  db.Store
	tokenMaker token.Maker
	router *gin.Engine
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TOKENSYMMETRICKEY)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config: config,
		store: store,
		tokenMaker: tokenMaker,
	}
	
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	server.setupRouter()

	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// Public routes
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)

	// Protected routes
	authRoutes := router.Group("/")
	authRoutes.Use(authMiddleware(server.tokenMaker))

	// Account
	authRoutes.POST("/accounts", server.createAccount)
	authRoutes.GET("/accounts/:id", server.getAccountById)
	authRoutes.GET("/accounts", server.listAccount)

	// Transfer
	authRoutes.POST("/transfers", server.createTransfer)

	server.router = router
}

// Start runs the HTTP server on a specific address
func (server *Server) Start(address string) error{
	return server.router.Run(address)
}	
func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
