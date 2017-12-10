package user

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/auth/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/gorouter"
)

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = errors.New("Empty request body")

// ErrInvalidURLParams is when an request has invalid or missing parameters.
var ErrInvalidURLParams = errors.New("Invalid request URL params")

// UserDomain stores services and data required for user domain to work correctly
type UserDomain struct {
	commandBus domain.CommandBus
	eventBus   domain.EventBus
	eventStore domain.EventStore
	jwt        jwt.Jwt
}

func (d *UserDomain) registerCommandHandlers() {
	repository := newRepository(fmt.Sprintf("%T", User{}), d.eventStore, d.eventBus)

	d.commandBus.Subscribe(RegisterWithEmail, onRegisterWithEmail(repository, d.jwt))
	d.commandBus.Subscribe(RegisterWithGoogle, onRegisterWithGoogle(repository))
	d.commandBus.Subscribe(RegisterWithFacebook, onRegisterWithFacebook(repository))
	d.commandBus.Subscribe(ChangeEmailAddress, onChangeEmailAddress(repository))
}

func (d *UserDomain) registerEventHandlers() {
	d.eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithEmail{}), onWasRegisteredWithEmail)
	d.eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithGoogle{}), onWasRegisteredWithGoogle)
	d.eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithFacebook{}), onWasRegisteredWithFacebook)
	d.eventBus.Subscribe(fmt.Sprintf("%T", &EmailAddressWasChanged{}), onEmailAddressWasChanged)
}

// AsRouter returns gorouter.Router instance
func (d *UserDomain) AsRouter() gorouter.Router {
	d.registerCommandHandlers()
	d.registerEventHandlers()

	router := gorouter.New()

	router.POST("/dispatch/{command}", d)
	router.USE(gorouter.POST, "/dispatch/"+ChangeEmailAddress, auth.Firewall("USER"))

	return router
}

// ServeHTTP implements http.Handler interface
func (d *UserDomain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var e error

	if r.Body == nil {
		r.WithContext(response.WithPayload(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   ErrEmptyRequestBody,
			Message: ErrEmptyRequestBody.Error(),
		}))
		return
	}

	params, ok := gorouter.FromContext(r.Context())
	if !ok {
		r.WithContext(response.WithPayload(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   ErrInvalidURLParams,
			Message: ErrInvalidURLParams.Error(),
		}))
		return
	}

	defer r.Body.Close()
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		r.WithContext(response.WithPayload(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid request body",
		}))
		return
	}

	out := make(chan error)
	defer close(out)

	go func() {
		d.commandBus.Publish(
			r.Context(),
			params.Value("command"),
			body,
			out,
		)
	}()

	if e = <-out; e != nil {
		r.WithContext(response.WithPayload(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid request",
		}))
		return
	}

	w.WriteHeader(http.StatusCreated)

	return
}

// NewDomain returns new user domain object allowing to register
// command and event handlers
// http routes as gorouter.Router
func NewDomain(cb domain.CommandBus, eb domain.EventBus, es domain.EventStore, j jwt.Jwt) *UserDomain {
	return &UserDomain{cb, eb, es, j}
}
