package router

import (
	http_cors "github.com/rs/cors"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	user_security "github.com/vardius/go-api-boilerplate/cmd/user/application/security"
	user_persistance "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence"
	http_recovery "github.com/vardius/go-api-boilerplate/pkg/http/recovery"
	http_response "github.com/vardius/go-api-boilerplate/pkg/http/response"
	http_authenticator "github.com/vardius/go-api-boilerplate/pkg/http/security/authenticator"
	log "github.com/vardius/go-api-boilerplate/pkg/log"
	gorouter "github.com/vardius/gorouter/v4"
)

// New provides new router
func New(logger *log.Logger, grpAuthClient auth_proto.AuthenticationServiceClient, repository user_persistance.UserRepository) gorouter.Router {
	auth := http_authenticator.NewToken(user_security.TokenAuthHandler(grpAuthClient, repository))

	http_recovery.WithLogger(logger)
	http_response.WithLogger(logger)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		http_cors.Default().Handler,
		http_response.WithXSS,
		http_response.WithHSTS,
		http_response.AsJSON,
		auth.FromHeader("USER"),
		auth.FromQuery("authToken"),
		http_recovery.WithRecover,
	)

	return router
}
