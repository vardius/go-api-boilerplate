package http

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/security/firewall"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	user_grpc "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	"github.com/vardius/gorouter"
)

// AddUserRoutes adds user routes to router
func AddUserRoutes(router gorouter.Router, grpClient user_proto.UserServiceClient) {
	subRouter := gorouter.New()

	subRouter.POST("/", buildListUserHandler(grpClient))

	subRouter.POST("/dispatch/{command}", buildCommandDispatchHandler(grpClient))
	subRouter.USE(gorouter.POST, "/dispatch/"+user_grpc.ChangeUserEmailAddress, firewall.GrantHTTPAccessFor("USER"))

	subRouter.POST("/{id}", buildCommandDispatchHandler(grpClient))

	// User domain
	router.Mount("/users", subRouter)
}

// buildCommandDispatchHandler wraps user gRPC client with http.Handler
func buildCommandDispatchHandler(userClient user_proto.UserServiceClient) http.Handler {
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
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request body"))
			return
		}

		_, e = userClient.DispatchCommand(r.Context(), &user_proto.DispatchCommandRequest{
			Name:    params.Value("command"),
			Payload: body,
		})
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request"))
			return
		}

		w.WriteHeader(http.StatusCreated)

		return
	}

	return http.HandlerFunc(fn)
}

// buildGetUserHandler wraps user gRPC client with http.Handler
func buildGetUserHandler(userClient user_proto.UserServiceClient) http.Handler {
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

		user, e := userClient.GetUser(r.Context(), &user_proto.GetUserRequest{
			Id: params.Value("id"),
		})
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request"))
			return
		}

		response.WithPayload(r.Context(), user)
		return
	}

	return http.HandlerFunc(fn)
}

// buildListUserHandler wraps user gRPC client with http.Handler
func buildListUserHandler(userClient user_proto.UserServiceClient) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), ErrEmptyRequestBody)
			return
		}

		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)

		resp, e := userClient.ListUsers(r.Context(), &user_proto.ListUserRequest{
			Page:  int32(math.Max(float64(page), 0)),
			Limit: int32(math.Max(float64(limit), 20)),
		})
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request"))
			return
		}

		response.WithPayload(r.Context(), resp)
		return
	}

	return http.HandlerFunc(fn)
}
