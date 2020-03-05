package http

import (
	"database/sql"
	"net/http"

	http_cors "github.com/rs/cors"
	"github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"

	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	user_security "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/security"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	user_persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http/handlers"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	http_middleware "github.com/vardius/go-api-boilerplate/internal/http/middleware"
	http_authenticator "github.com/vardius/go-api-boilerplate/internal/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/internal/http/middleware/firewall"
	http_metadata_middleware "github.com/vardius/go-api-boilerplate/internal/http/middleware/metadata"
	"github.com/vardius/go-api-boilerplate/internal/log"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

// NewRouter provides new router
func NewRouter(logger *log.Logger, repository user_persistence.UserRepository, commandBus commandbus.CommandBus, mysqlConnection *sql.DB, grpAuthClient auth_proto.AuthenticationServiceClient, grpcConnectionMap map[string]*grpc.ClientConn) gorouter.Router {
	auth := http_authenticator.NewToken(user_security.TokenAuthHandler(grpAuthClient, repository))

	// Global middleware
	router := gorouter.New(
		http_middleware.Recover(logger),
		http_metadata_middleware.WithMetadata(),
		http_middleware.Logger(logger),
		http_cors.Default().Handler,
		http_middleware.XSS(),
		http_middleware.HSTS(),
		http_middleware.Metrics(),
		http_middleware.LimitRequestBody(int64(10<<20)), // 10 MB is a lot of text.
		auth.FromHeader("USER"),
		auth.FromQuery("authToken"),
	)

	// Liveness probes are to indicate that your application is running
	router.GET("/v1/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/v1/readiness", handlers.BuildReadinessHandler(mysqlConnection, grpcConnectionMap))

	// Auth routes
	router.POST("/v1/auth/google", handlers.BuildSocialAuthHandler(commandBus, user.AuthUserWithProvider))
	router.POST("/v1/auth/facebook", handlers.BuildSocialAuthHandler(commandBus, user.AuthUserWithProvider))
	router.POST("/v1/google/callback", handlers.BuildSocialAuthHandler(commandBus, user.AuthUserWithProvider))
	router.POST("/v1/facebook/callback", handlers.BuildSocialAuthHandler(commandBus, user.AuthUserWithProvider))

	commandDispatchHandler := handlers.BuildCommandDispatchHandler(commandBus)

	// Public User routes
	router.POST("/v1/dispatch/{command}", commandDispatchHandler)
	// Protected User routes
	router.USE(http.MethodPost, "/v1/dispatch/"+user.ChangeUserEmailAddress, firewall.GrantAccessFor("USER"))

	router.GET("/v1/me", handlers.BuildMeHandler(repository))
	router.USE(http.MethodGet, "/v1/me", firewall.GrantAccessFor("USER"))

	router.GET("/v1/", handlers.BuildListUserHandler(repository))
	router.GET("/v1/{id}", handlers.BuildGetUserHandler(repository))

	return router
}
