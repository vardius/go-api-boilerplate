package http

import (
	"io/ioutil"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/security/firewall"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	user_grpc "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	"github.com/vardius/gorouter"
)

// AddUserRoutes adds user routes to router
func AddUserRoutes(router gorouter.Router, grpClient user_proto.UserClient) {
	httpClient := fromGRPC(grpClient)

	subRouter := gorouter.New()
	subRouter.POST("/dispatch/{command}", httpClient)
	subRouter.USE(gorouter.POST, "/dispatch/"+user_grpc.ChangeUserEmailAddress, firewall.GrantAccessFor("USER"))

	// User domain
	router.Mount("/users", subRouter)
}

// FromGRPC wraps user gRPC client with http.Handler
func fromGRPC(c user_proto.UserClient) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), ErrEmptyRequestBody)
			return
		}

		params, ok := gorouter.FromContext(r.Context())
		if !ok {
			response.WithError(r.Context(), ErrInvalidURLParams)
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, "Invalid request body", errors.INTERNAL))
			return
		}

		_, e = c.DispatchCommand(r.Context(), &user_proto.DispatchCommandRequest{
			Name:    params.Value("command"),
			Payload: body,
		})
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, "Invalid request", errors.INTERNAL))
			return
		}

		w.WriteHeader(http.StatusCreated)

		return
	}

	return http.HandlerFunc(fn)
}
