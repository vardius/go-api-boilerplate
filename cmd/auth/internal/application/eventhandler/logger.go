package eventhandler

import (
	"context"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/pkg/container"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

func GetLogger(ctx context.Context) *log.Logger {
	if requestContainer, ok := container.FromContext(ctx); ok {
		if v, ok := requestContainer.Get("logger"); ok {
			if logger, ok := v.(*log.Logger); ok {
				return logger
			}
		}
	}

	return log.New(config.Env.App.Environment)
}
