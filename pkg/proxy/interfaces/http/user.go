package http

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/firewall"
	"github.com/vardius/go-api-boilerplate/pkg/proxy/application/socialmedia"
	user_grpc_client "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/grpc"
	user_http_client "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/http"
	"github.com/vardius/go-api-boilerplate/pkg/user/application"
	"github.com/vardius/gorouter"
)

// AddUserRoutes adds user routes to router
func AddUserRoutes(router gorouter.Router, grpClient user_grpc_client.UserClient, jwtService jwt.Jwt) {
	httpClient := user_http_client.New(grpClient)

	// Routes
	// Social media auth routes
	router.POST("/auth/google/callback", socialmedia.NewGoogle(grpClient, jwtService))
	router.POST("/auth/facebook/callback", socialmedia.NewFacebook(grpClient, jwtService))
	// User domain
	router.Mount("/users", asSubRouter(httpClient))
}

func asSubRouter(h http.Handler) gorouter.Router {
	router := gorouter.New()

	router.POST("/dispatch/{command}", h)
	router.USE(gorouter.POST, "/dispatch/"+application.ChangeUserEmailAddress, firewall.GrantAccessFor("USER"))

	return router
}
