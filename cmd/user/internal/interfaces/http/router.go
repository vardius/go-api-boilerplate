package http

import (
	"database/sql"
	"net/http"

	httpcors "github.com/rs/cors"
	"github.com/vardius/gorouter/v4"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"

	httpformmiddleware "github.com/mar1n3r0/gorouter-middleware-formjson"

	usersecurity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/security"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http/handlers"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	httpmiddleware "github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

// NewRouter provides new router
func NewRouter(logger *log.Logger, claims auth.ClaimsProvider, repository userpersistence.UserRepository, commandBus commandbus.CommandBus, mysqlConnection *sql.DB, grpcConnectionMap map[string]*grpc.ClientConn, oauth2Config oauth2.Config, secretKey string) gorouter.Router {
	authenticator := httpauthenticator.NewToken(usersecurity.TokenAuthSecretHandler(claims))

	// Global middleware
	router := gorouter.New(
		httpmiddleware.Recover(logger),
		httpmiddleware.WithMetadata(),
		httpmiddleware.Logger(logger),
		httpmiddleware.WithContainer(),
		httpcors.Default().Handler,
		httpmiddleware.XSS(),
		httpmiddleware.HSTS(),
		httpmiddleware.Metrics(),
		httpmiddleware.LimitRequestBody(int64(10<<20)), // 10 MB is a lot of text.
		httpformmiddleware.FormJson(),
		authenticator.FromHeader("Restricted"),
		authenticator.FromQuery("authToken"),
		authenticator.FromCookie("at"),
	)

	// Liveness probes are to indicate that your application is running
	router.GET("/v1/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/v1/readiness", handlers.BuildReadinessHandler(mysqlConnection, grpcConnectionMap))

	// Auth routes
	router.POST("/v1/google/callback", handlers.BuildSocialAuthHandler(googleAPIURL, commandBus, user.RegisterUserWithGoogle, secretKey, oauth2Config))
	router.POST("/v1/facebook/callback", handlers.BuildSocialAuthHandler(facebookAPIURL, commandBus, user.RegisterUserWithFacebook, secretKey, oauth2Config))

	commandDispatchHandler := handlers.BuildCommandDispatchHandler(commandBus)

	// Public User routes
	router.POST("/v1/dispatch/{command}", commandDispatchHandler)
	// Protected User routes
	router.USE(http.MethodPost, "/v1/dispatch/"+user.ChangeUserEmailAddress, httpmiddleware.GrantAccessFor(identity.RoleUser))

	router.GET("/v1/me", handlers.BuildMeHandler(repository))
	router.USE(http.MethodGet, "/v1/me", httpmiddleware.GrantAccessFor(identity.RoleUser))

	router.GET("/v1/", handlers.BuildListUserHandler(repository))
	router.GET("/v1/{id}", handlers.BuildGetUserHandler(repository))

	return router
}
