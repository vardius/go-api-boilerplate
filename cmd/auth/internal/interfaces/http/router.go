package http

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"
	"gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http/handlers"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	httpmiddleware "github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// NewRouter provides new router
func NewRouter(
	logger golog.Logger,
	tokenAuthorizer auth.TokenAuthorizer,
	server *server.Server,
	commandBus commandbus.CommandBus,
	mysqlConnection *sql.DB,
	grpcConnectionMap map[string]*grpc.ClientConn,
	tokenRepository persistence.TokenRepository,
	clientRepository persistence.ClientRepository,
) http.Handler {
	authenticator := httpauthenticator.NewToken(tokenAuthorizer.Auth)

	// Global middleware
	router := gorouter.New(
		httpmiddleware.Recover(logger),
		httpmiddleware.WithMetadata(),
		httpmiddleware.Logger(logger),
		httpmiddleware.XSS(),
		httpmiddleware.HSTS(),
		authenticator.FromHeader("Restricted", logger),
		authenticator.FromQuery("authToken", logger),
		authenticator.FromCookie("at", logger),
		httpmiddleware.CORS(
			[]string{config.Env.App.Domain},
			config.Env.HTTP.Origins,
			config.Env.App.Environment == "development",
		),
		httpmiddleware.LimitRequestBody(int64(10<<20)),          // 10 MB is a lot of text.
		httpmiddleware.RateLimit(logger, 10, 10, 3*time.Minute), // 5 of requests per second with bursts of at most 10 requests
		httpmiddleware.Metrics(),
	)
	router.NotFound(response.NotFound())
	router.NotAllowed(response.NotAllowed())

	authorizeHandler := handlers.BuildAuthorizeHandler(server)
	router.GET("/authorize", authorizeHandler)
	router.POST("/authorize", authorizeHandler)
	router.POST("/token", handlers.BuildTokenHandler(server))

	router.POST("/dispatch/client/{command}", handlers.BuildClientCommandDispatchHandler(commandBus))
	router.POST("/dispatch/token/{command}", handlers.BuildTokenCommandDispatchHandler(commandBus))

	router.GET("/clients", handlers.BuildListClientsHandler(clientRepository))
	router.GET("/clients/{clientID}", handlers.BuildGetClientHandler(clientRepository))
	router.GET("/clients/{clientID}/tokens", handlers.BuildListTokensHandler(tokenRepository, clientRepository))
	router.GET("/users/{userID}/tokens", handlers.BuildListUserAuthTokensHandler(tokenRepository))

	// middleware applies to whole subtrees
	router.USE(http.MethodGet, "/users", httpmiddleware.GrantAccessFor(identity.RoleUser))
	router.USE(http.MethodGet, "/clients", httpmiddleware.GrantAccessFor(identity.RoleUser))
	router.USE(http.MethodPost, "/dispatch", httpmiddleware.GrantAccessFor(identity.RoleUser))

	mainRouter := gorouter.New()
	mainRouter.NotFound(response.NotFound())
	mainRouter.NotAllowed(response.NotAllowed())

	// We do not want to apply middleware for this handlers
	// Liveness probes are to indicate that your application is running
	mainRouter.GET("/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	mainRouter.GET("/readiness", handlers.BuildReadinessHandler(mysqlConnection, grpcConnectionMap))

	mainRouter.Mount("/v1", router)

	return mainRouter
}
