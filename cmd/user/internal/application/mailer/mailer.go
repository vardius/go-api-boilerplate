package mailer

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/email"
)

const (
	FROM = "noreply@go-api-boilerplate.local"
)

func SendLoginEmail(ctx context.Context, to string, authToken string) error {
	var template bytes.Buffer
	if err := email.Login.Execute(&template, struct {
		Title    string
		LoginURL string
	}{
		Title:    "Login to go-api-boilerplate",
		LoginURL: "https://go-api-boilerplate.local?authToken=" + authToken,
	}); err != nil {
		return err
	}

	return send(FROM, []string{to}, template.Bytes())
}

func send(from string, to []string, msg []byte) error {
	if from == "" {
		from = FROM
	}

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", config.Env.Mailer.Host, config.Env.Mailer.Port),
		smtp.PlainAuth("", config.Env.Mailer.User, config.Env.Mailer.Password, config.Env.Mailer.Host),
		from,
		to,
		msg,
	)
}
