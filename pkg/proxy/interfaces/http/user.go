package http

import (
	"github.com/vardius/go-api-boilerplate/pkg/common/security/firewall"
	proxy_user_http "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/http"
	user_grpc "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/proto"
	"github.com/vardius/gorouter"
)

// AddUserRoutes adds user routes to router
func AddUserRoutes(router gorouter.Router, grpClient user_proto.UserClient) {
	httpClient := proxy_user_http.FromGRPC(grpClient)

	subRouter := gorouter.New()
	subRouter.POST("/dispatch/{command}", httpClient)
	subRouter.USE(gorouter.POST, "/dispatch/"+user_grpc.ChangeUserEmailAddress, firewall.GrantAccessFor("USER"))

	// User domain
	router.Mount("/users", subRouter)
}
