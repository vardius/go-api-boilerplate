package main

import (
	"context"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	config "github.com/vardius/go-api-boilerplate/cmd/test/config"
	application "github.com/vardius/go-api-boilerplate/internal/application"
	buildinfo "github.com/vardius/go-api-boilerplate/internal/buildinfo"
	log "github.com/vardius/go-api-boilerplate/internal/log"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.App.Environment)
	router := NewRouter(logger)
	app := application.New(logger)

	app.AddAdapters(
		NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.HTTP.Host, config.Env.HTTP.Port),
			router,
		),
	)

	if config.Env.App.Environment == "development" {
		app.AddAdapters(
			application.NewDebugAdapter(
				fmt.Sprintf("%s:%d", config.Env.Debug.Host, config.Env.Debug.Port),
			),
		)
	}

	app.WithShutdownTimeout(config.Env.App.ShutdownTimeout)
	app.Run(ctx)
}
