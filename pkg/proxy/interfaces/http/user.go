package http

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/security/firewall"
	user_grpc_client "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/grpc"
	user_http_client "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/http"
	user_grpc_server "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	"github.com/vardius/gorouter"
)

// AddUserRoutes adds user routes to router
func AddUserRoutes(router gorouter.Router, grpClient user_grpc_client.UserClient) {
	httpClient := user_http_client.New(grpClient)

	// User domain
	router.Mount("/users", asSubRouter(httpClient))
}

func asSubRouter(h http.Handler) gorouter.Router {
	router := gorouter.New()

	router.POST("/dispatch/{command}", h)
	router.USE(gorouter.POST, "/dispatch/"+user_grpc_server.ChangeUserEmailAddress, firewall.GrantAccessFor("USER"))

	return router
}
