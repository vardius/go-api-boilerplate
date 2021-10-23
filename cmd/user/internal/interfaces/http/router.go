package http

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http/handlers"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	httpmiddleware "github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/http/response/json"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/gorouter/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

// NewRouter provides new router
func NewRouter(
	cfg *config.Config,
	tokenAuthorizer auth.TokenAuthorizer,
	repository userpersistence.UserRepository,
	commandBus commandbus.CommandBus,
	sqlConn *sql.DB, mongoConn *mongo.Client,
	grpcConnectionMap map[string]*grpc.ClientConn,
) http.Handler {
	authenticator := httpauthenticator.NewToken(tokenAuthorizer.Auth)

	// Global middleware
	router := gorouter.New(
		httpmiddleware.Recover(),
		httpmiddleware.WithMetadata(),
		httpmiddleware.Logger(),
		httpmiddleware.XSS(),
		httpmiddleware.HSTS(),
		authenticator.FromHeader("Restricted"),
		authenticator.FromQuery("authToken"),
		authenticator.FromCookie("at"),
		httpmiddleware.CORS(
			cfg.HTTP.Origins,
			cfg.App.Environment == "development",
		),
		httpmiddleware.LimitRequestBody(int64(10<<20)), // 10 MB is a lot of text.
		httpmiddleware.Metrics(),
		httpmiddleware.RateLimit(10, 10, 3*time.Minute), // 5 of requests per second with bursts of at most 10 requests
	)
	router.NotFound(json.NotFound())
	router.NotAllowed(json.NotAllowed())

	router.GET("/", handlers.BuildListUserHandler(repository))
	router.GET("/me", handlers.BuildMeHandler(repository))
	router.GET("/{id}", handlers.BuildGetUserHandler(repository))
	router.POST("/dispatch/user/{command}", handlers.BuildUserCommandDispatchHandler(commandBus))

	var googleOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/v1/google/callback", cfg.App.ApiBaseURL),
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
	router.POST("/google", handlers.BuildSocialAuthHandler(googleOauthConfig))
	router.POST("/google/callback", handlers.BuildAuthCallbackHandler(googleOauthConfig, googleAPIURL, commandBus, user.RegisterUserWithGoogle))

	var facebookOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/v1/facebook/callback", cfg.App.ApiBaseURL),
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		Scopes:       []string{"public_profile"},
		Endpoint:     facebook.Endpoint,
	}
	router.POST("/facebook", handlers.BuildSocialAuthHandler(facebookOauthConfig))
	router.POST("/facebook/callback", handlers.BuildAuthCallbackHandler(facebookOauthConfig, facebookAPIURL, commandBus, user.RegisterUserWithGoogle))

	router.USE(http.MethodGet, "/me", httpmiddleware.GrantAccessFor(identity.PermissionUserRead))
	router.USE(http.MethodPost, "/dispatch/"+user.ChangeUserEmailAddress, httpmiddleware.GrantAccessFor(identity.PermissionUserWrite))

	mainRouter := gorouter.New()
	mainRouter.NotFound(json.NotFound())
	mainRouter.NotAllowed(json.NotAllowed())

	// We do not want to apply middleware for this handlers
	// Liveness probes are to indicate that your application is running
	mainRouter.GET("/health", handlers.BuildLivenessHandler())
	// Readiness is meant to check if your application is ready to serve traffic
	mainRouter.GET("/readiness", handlers.BuildReadinessHandler(sqlConn, mongoConn, grpcConnectionMap))

	mainRouter.Mount("/v1", router)

	return mainRouter
}
