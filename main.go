package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/edr3x/gateway-impl/internal/api/handlers"
	pb "github.com/edr3x/gateway-impl/pkg/proto"
)

var logger *zap.Logger

func init() {
	logger = createProductionLogger()
	if os.Getenv("ENV") == "dev" {
		logger = zap.Must(zap.NewDevelopment())
	}
	zap.ReplaceGlobals(logger)
	grpc_zap.ReplaceGrpcLogger(logger)
}

func main() {
	ctx := context.Background()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	grpcAddress := "0.0.0.0:" + grpcPort
	listener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		logger.Fatal("failed to start grpc server:", zap.Error(err))
	}

	recoveryFunc := func(p any) (err error) {
		return status.Errorf(codes.Internal, "server panicked: %v", p)
	}
	opts := []recovery.Option{
		recovery.WithRecoveryHandler(recoveryFunc),
	}

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(opts...),
		grpc_zap.UnaryServerInterceptor(logger),
	))

	pb.RegisterAuthServiceServer(s, handlers.NewAuthHandler())
	logger.Info(fmt.Sprintf("Server started at %s", listener.Addr()))
	go func() {
		if err := s.Serve(listener); err != nil {
			logger.Fatal("Grpc server starting error", zap.Error(err))
		}
	}()

	conn, err := grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("Error dialing", zap.Error(err))
	}
	defer conn.Close()

	mux := runtime.NewServeMux()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	gatewayServer := &http.Server{
		Handler:      mux,
		Addr:         "0.0.0.0:" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := pb.RegisterAuthServiceHandler(ctx, mux, conn); err != nil {
		logger.Fatal("", zap.Error(err))
	}

	c, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger.Info(fmt.Sprintf("Listening on port %s", port))
	go func() {
		if err := gatewayServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(err.Error())
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-c.Done()
	if err := gatewayServer.Shutdown(ctx); err != nil {
		logger.Fatal(err.Error())
	}
}

func createProductionLogger() *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
	}
	return zap.Must(config.Build())
}
