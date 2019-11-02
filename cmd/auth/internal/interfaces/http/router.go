package http

import (
	"database/sql"

	http_cors "github.com/rs/cors"
	handlers "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http/handlers"
	http_middleware "github.com/vardius/go-api-boilerplate/internal/http/middleware"
	http_metadata_middleware "github.com/vardius/go-api-boilerplate/internal/http/middleware/metadata"
	log "github.com/vardius/go-api-boilerplate/internal/log"
	gorouter "github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"
	"gopkg.in/oauth2.v3/server"
)

// NewRouter provides new router
func NewRouter(logger *log.Logger, server *server.Server, mysqlConnection *sql.DB, grpcConnectionMap map[string]*grpc.ClientConn) gorouter.Router {
	// Global middleware
	router := gorouter.New(
		http_metadata_middleware.WithMetadata(),
		http_middleware.Logger(logger),
		http_middleware.LimitRequestBody(int64(10<<20)), // 10 MB is a lot of text.
		http_cors.Default().Handler,
		http_middleware.XSS(),
		http_middleware.HSTS(),
		http_middleware.Metrics(),
		http_middleware.Recover(logger),
	)

	// Liveness probes are to indicate that your application is running
	router.GET("/v1/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/v1/readiness", handlers.BuildReadinessHandler(mysqlConnection, grpcConnectionMap))

	router.POST("/v1/authorize", handlers.BuildAuthorizeHandler(server))
	router.POST("/v1/token", handlers.BuildTokenHandler(server))

	router.Compile()

	return router
}
