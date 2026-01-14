package main

import (
	"context"
	"embed"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"io/fs"
	"log"
	"net"
	"net/http"
	"simple-bank/api"
	db "simple-bank/db/sqlc"
	"simple-bank/db/util"
	"simple-bank/gapi"
	"simple-bank/pb"
	"strings"
	// _"simple-bank/doc/statik"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
)

//go:embed doc/swagger/*
var swaggerFS embed.FS

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Create connection pool
	pool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer pool.Close()

	runDBMigration(config.MigrationURL, config.DBSource)

	// Initialize store
	store := db.NewStore(pool)
	go runGatewayServer(config, store)
	runGrpcServer(config, store)
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal("cannot create new migrate instance:", err)
	}
	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("failed to run migrate up:", err)
	}
	log.Println("db migrated successfully")	
}

func runGrpcServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	log.Printf("start gRPC server at %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot start gRPC server:", err)
	}
}

func runGatewayServer(config util.Config, store db.Store) {
	// Initialize gRPC server
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// gRPC-Gateway mux
	grpcMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			case "user-agent":
				return "grpcgateway-user-agent", true
			case "x-forwarded-for":
				return "x-forwarded-for", true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	ctx := context.Background()
	if err := pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server); err != nil {
		log.Fatal("cannot register gRPC-Gateway handler:", err)
	}

	// HTTP mux
	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	swaggerSubFS, err := fs.Sub(swaggerFS, "doc/swagger")
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle("/swagger/",
		http.StripPrefix("/swagger/",
			http.FileServer(http.FS(swaggerSubFS)),
		),
	)

	mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// Start listener
	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	log.Printf("HTTP gateway started at %s", listener.Addr().String())
	if err := http.Serve(listener, mux); err != nil {
		log.Fatal("cannot start HTTP gateway:", err)
	}
}

func runGinServer(config util.Config, store db.Store) {
	// Initialize server ✅
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// Start HTTP server
	log.Printf("Starting server on %s", config.HTTPServerAddress)
	if err := server.Start(config.HTTPServerAddress); err != nil {
		log.Fatal("cannot start server:", err)
	}

	log.Printf("Go sees TOKEN_SYMMETRIC_KEY length: %d", len(config.TokenSymmetricKey))
}
