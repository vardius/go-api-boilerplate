package http

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	appidentity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http/handlers"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	pkgauth "github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	httpmiddleware "github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

// NewRouter provides new router
func NewRouter(logger golog.Logger,
	tokenAuthorizer auth.TokenAuthorizer,
	repository userpersistence.UserRepository,
	commandBus commandbus.CommandBus,
	tokenProvider pkgauth.TokenProvider,
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
	router.POST("/dispatch/user/{command}", handlers.BuildUserCommandDispatchHandler(commandBus))

	var googleOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/v1/google/callback", config.Env.App.ApiBaseURL),
		ClientID:     config.Env.Google.ClientID,
		ClientSecret: config.Env.Google.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
	router.POST("/google", handlers.BuildSocialAuthHandler(googleOauthConfig))
	router.POST("/google/callback", handlers.BuildAuthCallbackHandler(googleOauthConfig, googleAPIURL, commandBus, user.RegisterUserWithGoogle, tokenProvider, identityProvider))

	var facebookOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/v1/facebook/callback", config.Env.App.ApiBaseURL),
		ClientID:     config.Env.Google.ClientID,
		ClientSecret: config.Env.Google.ClientSecret,
		Scopes:       []string{"public_profile"},
		Endpoint:     facebook.Endpoint,
	}
	router.POST("/facebook", handlers.BuildSocialAuthHandler(facebookOauthConfig))
	router.POST("/facebook/callback", handlers.BuildAuthCallbackHandler(facebookOauthConfig, facebookAPIURL, commandBus, user.RegisterUserWithGoogle, tokenProvider, identityProvider))

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
