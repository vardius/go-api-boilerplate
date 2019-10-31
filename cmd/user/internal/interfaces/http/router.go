package http

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	http_cors "github.com/rs/cors"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/proto"
	user_security "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/security"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	user_persistance "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	handlers "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http/handlers"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	http_recovery "github.com/vardius/go-api-boilerplate/pkg/http/recovery"
	http_response "github.com/vardius/go-api-boilerplate/pkg/http/response"
	http_authenticator "github.com/vardius/go-api-boilerplate/pkg/http/security/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/http/security/firewall"
	log "github.com/vardius/go-api-boilerplate/pkg/log"
	gorouter "github.com/vardius/gorouter/v4"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

// NewRouter provides new router
func NewRouter(logger *log.Logger, repository user_persistance.UserRepository, commandBus commandbus.CommandBus, mysqlConnection *sql.DB, grpAuthClient auth_proto.AuthenticationServiceClient, grpcConnectionMap map[string]*grpc.ClientConn, oauth2Config oauth2.Config, secretKey string) gorouter.Router {
	auth := http_authenticator.NewToken(user_security.TokenAuthHandler(grpAuthClient, repository))

	http_recovery.WithLogger(logger)
	http_response.WithLogger(logger)

	// Global middleware
	router := goroter.New(
		logger.LogRequest,
		http_cors.Default().Handler,
		http_response.WithXSS,
		http_response.WithHSTS,
		auth.FromHeader("USER"),
		auth.FromQuery("authToken"),
		http_recovery.WithRecover,
	)

	// Liveness probes are to indicate that your application is running
	router.GET("/v1/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/v1/readiness", handlers.BuildReadinessHandler(mysqlConnection, grpcConnectionMap))

	// Auth routes
	router.POST("/v1/google/callback", handlers.BuildSocialAuthHandler(googleAPIURL, commandBus, user.RegisterUserWithGoogle, secretKey, oauth2Config))
	router.POST("/v1/facebook/callback", handlers.BuildSocialAuthHandler(facebookAPIURL, commandBus, user.RegisterUserWithFacebook, secretKey, oauth2Config))

	// User routes
	router.POST("/v1/dispatch/{command}", handlers.BuildCommandDispatchHandler(commandBus))
	router.USE(http.MethodPost, "/v1/dispatch/"+user.ChangeUserEmailAddress, firewall.GrantAccessFor("USER"))

	router.GET("/v1/me", handlers.BuildMeHandler(repository))
	router.USE(http.MethodGet, "/v1/me", firewall.GrantAccessFor("USER"))

	router.GET("/v1/", handlers.BuildListUserHandler(repository))
	router.GET("/v1/{id}", handlers.BuildGetUserHandler(repository))

	return router
}
