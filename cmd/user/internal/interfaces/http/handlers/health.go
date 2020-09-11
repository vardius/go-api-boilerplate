package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"

	"google.golang.org/grpc"

	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
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
			appErr := apperrors.Wrap(err)

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		for name, conn := range connMap {
			if !grpcutils.IsConnectionServing(r.Context(), name, conn) {
				appErr := apperrors.New(fmt.Sprintf("gRPC connection %s is not serving", name))

				response.MustJSONError(r.Context(), w, appErr)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}

	return http.HandlerFunc(fn)
}
