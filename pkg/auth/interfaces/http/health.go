package http

import (
	"context"
	"net/http"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// AddHealthCheckRoutes adds health checks route
func AddHealthCheckRoutes(router gorouter.Router, log golog.Logger, ac *grpc.ClientConn, uc *grpc.ClientConn) {
	// Liveness probes are to indicate that your application is running
	router.GET("/healthz", buildLivenessHandler(log))
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/readiness", buildReadinessHandler(log, ac, uc))
}

func buildLivenessHandler(log golog.Logger) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}

	return http.HandlerFunc(fn)
}

func buildReadinessHandler(log golog.Logger, ac *grpc.ClientConn, uc *grpc.ClientConn) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		status := getStatusCodeFromGRPConnectionHealthCheck(r.Context(), log, ac, "auth")
		if status != 200 {
			w.WriteHeader(status)
			return
		}

		status = getStatusCodeFromGRPConnectionHealthCheck(r.Context(), log, uc, "user")
		if status != 200 {
			w.WriteHeader(status)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}

	return http.HandlerFunc(fn)
}

func getStatusCodeFromGRPConnectionHealthCheck(ctx context.Context, log golog.Logger, conn *grpc.ClientConn, service string) int {
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{Service: service})
	if err != nil {
		if stat, ok := status.FromError(err); ok {
			log.Warning(ctx, "error %d: health rpc failed: %+v", stat.Code(), err)
		} else {
			log.Warning(ctx, "error: health rpc failed: %+v", err)
		}

		return 500
	}

	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {

		return 500
	}

	return 200
}
