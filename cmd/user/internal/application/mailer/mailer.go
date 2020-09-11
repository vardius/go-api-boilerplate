package mailer

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"net/url"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/email"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

const (
	FROM = "noreply@go-api-boilerplate.local"
)

// @TODO @FIXME: remove usage
// is to avid plain auth error if !server.TLS { return "", nil, errors.New("unencrypted connection") }
type unencryptedAuth struct {
	smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	s := *server
	s.TLS = true
	return a.Auth.Start(&s)
}

func SendLoginEmail(ctx context.Context, subject, to string, authToken, redirectPath string) error {
	var template bytes.Buffer
	if err := email.Login.Execute(&template, struct {
		Title    string
		LoginURL string
	}{
		Title: "Login to go-api-boilerplate",
		LoginURL: fmt.Sprintf("%s?%s", config.Env.App.Domain, url.Values{
			"r":         []string{redirectPath},
			"authToken": []string{authToken},
		}.Encode()),
	}); err != nil {
		return apperrors.Wrap(err)
	}

	return sendHTMLEmail(subject, FROM, []string{to}, template.Bytes())
}

func sendHTMLEmail(subject, from string, to []string, body []byte) error {
	if from == "" {
		from = FROM
	}

	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";"
	source := []byte(fmt.Sprintf("Subject: %s\n%s\n\n", subject, mime))

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", config.Env.Mailer.Host, config.Env.Mailer.Port),
		unencryptedAuth{smtp.PlainAuth("", config.Env.Mailer.User, config.Env.Mailer.Password, config.Env.Mailer.Host)},
		from,
		to,
		append(source, body...),
	)
}
