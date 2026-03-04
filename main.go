package main

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"simple-bank/api"
	db "simple-bank/db/sqlc"
	"simple-bank/db/util"
	"simple-bank/gapi"
	"simple-bank/pb"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	// _"simple-bank/doc/statik"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"simple-bank/kf"
	"simple-bank/mq"
	"simple-bank/worker"

	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
)

//go:embed doc/swagger/*
var swaggerFS embed.FS

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Msg("cannot load config")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	pool, _ := pgxpool.New(context.Background(), config.DBSource)
	defer pool.Close()

	runDBMigration(config.MigrationURL, config.DBSource)

	store := db.NewStore(pool)

	kafkaConfig := kf.LoadConfig()
	if config.Environment == "development" {
		kafkaConfig.Brokers = []string{"simple-bank-kafka:9092"}
	}

	kafkaPublisher, err := kf.NewKafkaPublisher(kafkaConfig.Brokers)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create kafka publisher")
	}
	defer kafkaPublisher.Close()

	outboxRelay := worker.NewOutboxRelay(
		store,
		kafkaPublisher,
		kafkaConfig.Topic,
		3*time.Second,
	)
	go outboxRelay.Start()

	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}

	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)

	conn, ch := mq.NewRabbitMQ(config.RabbitMQURL)
	defer conn.Close()
	defer ch.Close()

	mq.StartUserCreatedConsumer(ch, store)

	go runTaskProcessor(redisOpt, store)
	go runGatewayServer(config, store, taskDistributor, ch)
	runGrpcServer(config, store, taskDistributor, ch)

	// runGinServer(config, store)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Info().Msg("📊 Metrics server starting at :9091")
		log.Fatal().Err(http.ListenAndServe(":9091", mux)).Msg("metrics server failed")
	}()
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Msg("cannot create new migrate instance:")
	}
	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Msg("failed to run migrate up:")
	}
	log.Info().Msg("db migrated successfully")
}

func runTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store) {
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store)
	log.Info().Msg("start task processor")
	err := taskProcessor.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start task processor:")
	}
}
func runGrpcServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor, rabbitCh *amqp.Channel) {
	server, err := gapi.NewServer(config, store, taskDistributor, rabbitCh)
	if err != nil {
		log.Fatal().Msg("cannot create server:")
	}

	grpcLogger := grpc.UnaryInterceptor(gapi.GrpcLogger)

	grpcServer := grpc.NewServer(grpcLogger)
	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal().Msg("cannot create listener:")
	}

	log.Info().Msgf("start gRPC server at %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal().Msg("cannot start gRPC server:")
	}
}

func runGatewayServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor, rabbitCh *amqp.Channel) {
	// Initialize gRPC server
	server, err := gapi.NewServer(config, store, taskDistributor, rabbitCh)
	if err != nil {
		log.Fatal().Msg("cannot create server:")
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
		log.Fatal().Msg("cannot register gRPC-Gateway handler:")
	}

	// HTTP mux
	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	swaggerSubFS, err := fs.Sub(swaggerFS, "doc/swagger")
	if err != nil {
		log.Fatal().Msg("cannot create sub fs:")
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
		log.Fatal().Msg("cannot create listener:")
	}

	log.Info().Msgf("HTTP gateway started at %s", listener.Addr().String())
	handler := gapi.HttpLogger(mux)
	if err := http.Serve(listener, handler); err != nil {
		log.Fatal().Msg("cannot start HTTP gateway:")
	}
}

func runGinServer(config util.Config, store db.Store) {
	// Initialize server ✅
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Msg("cannot create server:")
	}

	// Start HTTP server
	log.Info().Msgf("Starting server on %s", config.HTTPServerAddress)
	if err := server.Start(config.HTTPServerAddress); err != nil {
		log.Fatal().Msg("cannot start server:")
	}

	log.Info().Msgf("Go sees TOKEN_SYMMETRIC_KEY length: %d", len(config.TokenSymmetricKey))
}
