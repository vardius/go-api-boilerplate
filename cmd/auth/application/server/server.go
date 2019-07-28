package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"
	"github.com/vardius/shutdown"
	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

// Server interface
type Server interface {
	Run(ctx context.Context, router gorouter.Router, grpcServer *grpc.Server, grpcHealthServer *grpc_health.Server)
}

type server struct {
	logger      golog.Logger
	httpAddress string
	tcpAddress  string
}

// New provides new server
func New(logger golog.Logger, httpAddress, tcpAddress string) Server {
	return &server{
		logger:      logger,
		httpAddress: httpAddress,
		tcpAddress:  tcpAddress,
	}
}

func (s *server) Run(ctx context.Context, router gorouter.Router, grpcServer *grpc.Server, grpcHealthServer *grpc_health.Server) {
	srv := &http.Server{
		Addr:         s.httpAddress,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	lis, err := net.Listen("tcp", s.tcpAddress)
	if err != nil {
		s.logger.Critical(ctx, "tcp failed to listen %s\n%v\n", s.tcpAddress, err)
		os.Exit(1)
	}

	stop := func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		s.logger.Info(ctx, "shutting down...\n")

		grpcServer.GracefulStop()

		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Critical(ctx, "shutdown error: %v\n", err)
		} else {
			s.logger.Info(ctx, "gracefully stopped\n")
		}
	}

	go func() {
		s.logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
		stop()
		os.Exit(1)
	}()

	go func() {
		s.logger.Critical(ctx, "%v\n", srv.ListenAndServe())
		stop()
		os.Exit(1)
	}()

	grpcHealthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_SERVING)

	s.logger.Info(ctx, "tcp running at %s\n", s.tcpAddress)
	s.logger.Info(ctx, "http running at %s\n", s.httpAddress)

	shutdown.GracefulStop(stop)
}
