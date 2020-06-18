package http

import (
	"database/sql"
	"time"

	httpcors "github.com/rs/cors"
	"github.com/vardius/gocontainer"
	"github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"
	"gopkg.in/oauth2.v4/server"

	httpformmiddleware "github.com/mar1n3r0/gorouter-middleware-formjson"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http/handlers"
	httpmiddleware "github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// NewRouter provides new router
func NewRouter(logger *log.Logger, server *server.Server, mysqlConnection *sql.DB, grpcConnectionMap map[string]*grpc.ClientConn) gorouter.Router {
	// Global middleware
	router := gorouter.New(
		httpmiddleware.Recover(logger),
		httpmiddleware.WithMetadata(),
		httpmiddleware.Logger(logger),
		httpmiddleware.WithContainer(gocontainer.New()), // used to pass logger between middleware
		httpcors.Default().Handler,
		httpmiddleware.XSS(),
		httpmiddleware.HSTS(),
		httpmiddleware.Metrics(),
		httpmiddleware.LimitRequestBody(int64(10<<20)),          // 10 MB is a lot of text.
		httpmiddleware.RateLimit(logger, 10, 10, 3*time.Minute), // 5 of requests per second with bursts of at most 10 requests
		httpformmiddleware.FormJson(),
	)

	// Liveness probes are to indicate that your application is running
	router.GET("/v1/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/v1/readiness", handlers.BuildReadinessHandler(mysqlConnection, grpcConnectionMap))

	router.POST("/v1/authorize", handlers.BuildAuthorizeHandler(server))
	router.POST("/v1/token", handlers.BuildTokenHandler(server))

	return router
}
