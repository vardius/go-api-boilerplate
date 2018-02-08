package userclient

import (
	"fmt"
	"context"
	"github.com/vardius/go-api-boilerplate/internal/user"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/security/firewall"
	pb "github.com/vardius/go-api-boilerplate/rpc/domain"
	"github.com/vardius/gorouter"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
)

// UserClient interface
type UserClient interface {
	DispatchAndClose(ctx context.Context, command string, payload []byte) error
	AsRouter() gorouter.Router
}

type userClient struct {
	host string
	port int
}

// DispatchAndClose dials user domain server and dispatches command
// then closes connection
func (c *userClient) DispatchAndClose(ctx context.Context, command string, payload []byte) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", c.host, c.port), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewDomainClient(conn)
	_, err = client.Dispatch(ctx, &pb.Command{
		Name:    command,
		Payload: payload,
	})

	return err
}

// AsRouter returns gorouter.Router instance
func (c *userClient) AsRouter() gorouter.Router {
	router := gorouter.New()

	router.POST("/dispatch/{command}", c)
	router.USE(gorouter.POST, "/dispatch/"+user.ChangeEmailAddress, firewall.GrantAccessFor("USER"))

	return router
}

// ServeHTTP implements http.Handler interface
func (c *userClient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var e error

	if r.Body == nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   ErrEmptyRequestBody,
			Message: ErrEmptyRequestBody.Error(),
		})
		return
	}

	params, ok := gorouter.FromContext(r.Context())
	if !ok {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   ErrInvalidURLParams,
			Message: ErrInvalidURLParams.Error(),
		})
		return
	}

	defer r.Body.Close()
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid request body",
		})
		return
	}

	e = c.DispatchAndClose(r.Context(), params.Value("command"), body)
	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid request",
		})
		return
	}

	w.WriteHeader(http.StatusCreated)

	return
}

// New creates new user client
func New(host string, port int) UserClient {
	return &userClient{host, port}
}
