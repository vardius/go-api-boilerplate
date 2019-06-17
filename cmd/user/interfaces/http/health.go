package http

import (
	"database/sql"
	"net/http"

	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"
)

// AddHealthCheckRoutes adds health checks route
func AddHealthCheckRoutes(router gorouter.Router, db *sql.DB, connMap map[string]*grpc.ClientConn) {
	// Liveness probes are to indicate that your application is running
	router.GET("/healthz", buildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/readiness", buildReadinessHandler(db, connMap))
}

func buildLivenessHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}

	return http.HandlerFunc(fn)
}

func buildReadinessHandler(db *sql.DB, connMap map[string]*grpc.ClientConn) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(500)
			return
		}

		for name, conn := range connMap {
			if !grpc_utils.IsConnectionServing(name, conn) {
				w.WriteHeader(500)
				return
			}
		}

		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}

	return http.HandlerFunc(fn)
}
