package http

import (
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/proxy/interfaces/http/auth"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	"github.com/vardius/gorouter"
)

// AddAuthRoutes adds user routes to router
func AddAuthRoutes(router gorouter.Router, grpClient user_proto.UserClient, jwtService jwt.Jwt) {
	// Social media auth routes
	router.POST("/auth/google/callback", auth.NewGoogle(grpClient, jwtService))
	router.POST("/auth/facebook/callback", auth.NewFacebook(grpClient, jwtService))
}
