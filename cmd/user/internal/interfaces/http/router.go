package http

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	appidentity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http/handlers"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	httpmiddleware "github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

// NewRouter provides new router
func NewRouter(logger *log.Logger,
	tokenAuthorizer auth.TokenAuthorizer,
	repository userpersistence.UserRepository,
	commandBus commandbus.CommandBus,
	tokenProvider oauth2.TokenProvider,
	mysqlConnection *sql.DB,
	identityProvider appidentity.Provider,
	grpcConnectionMap map[string]*grpc.ClientConn,
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

	router.GET("/", handlers.BuildListUserHandler(repository))
	router.GET("/me", handlers.BuildMeHandler(repository))
	router.GET("/{id}", handlers.BuildGetUserHandler(repository))
	router.POST("/google/callback", handlers.BuildSocialAuthHandler(googleAPIURL, commandBus, user.RegisterUserWithGoogle, tokenProvider, identityProvider))
	router.POST("/facebook/callback", handlers.BuildSocialAuthHandler(facebookAPIURL, commandBus, user.RegisterUserWithFacebook, tokenProvider, identityProvider))
	router.POST("/dispatch/user/{command}", handlers.BuildUserCommandDispatchHandler(commandBus))

	router.USE(http.MethodGet, "/me", httpmiddleware.GrantAccessFor(identity.RoleUser))
	router.USE(http.MethodPost, "/dispatch/"+user.ChangeUserEmailAddress, httpmiddleware.GrantAccessFor(identity.RoleUser))

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
