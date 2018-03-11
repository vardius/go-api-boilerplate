package http

import (
	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/proxy/application/socialmedia"
	user_grpc_client "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/grpc"
	"github.com/vardius/gorouter"
)

// AddAuthRoutes adds user routes to router
func AddAuthRoutes(router gorouter.Router, grpClient user_grpc_client.UserClient, jwtService jwt.Jwt) {
	// Social media auth routes
	router.POST("/auth/google/callback", socialmedia.NewGoogle(grpClient, jwtService))
	router.POST("/auth/facebook/callback", socialmedia.NewFacebook(grpClient, jwtService))
}
