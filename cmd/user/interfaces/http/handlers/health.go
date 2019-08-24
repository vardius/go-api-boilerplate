package handlers

import (
	"database/sql"
	"net/http"

	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"google.golang.org/grpc"
)

// BuildLivenessHandler provides liveness handler
func BuildLivenessHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}

	return http.HandlerFunc(fn)
}

// BuildReadinessHandler provides readiness handler
func BuildReadinessHandler(db *sql.DB, connMap map[string]*grpc.ClientConn) http.Handler {
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
