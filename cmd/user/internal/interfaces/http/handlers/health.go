package handlers

import (
	"database/sql"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"

	"google.golang.org/grpc"

	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
)

// BuildLivenessHandler provides liveness handler
func BuildLivenessHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}

	return http.HandlerFunc(fn)
}

// BuildReadinessHandler provides readiness handler
func BuildReadinessHandler(db *sql.DB, connMap map[string]*grpc.ClientConn) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			appErr := errors.Wrap(err, errors.INTERNAL, "Database is not responding")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		for name, conn := range connMap {
			if !grpc_utils.IsConnectionServing(r.Context(), name, conn) {
				appErr := errors.Newf(errors.INTERNAL, "gRPC connection %s is not serving", name)
				w.WriteHeader(errors.HTTPStatusCode(appErr))

				if err := response.JSON(r.Context(), w, appErr); err != nil {
					panic(err)
				}
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}

	return http.HandlerFunc(fn)
}
