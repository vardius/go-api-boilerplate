package router

import (
	http_cors "github.com/rs/cors"
	http_recovery "github.com/vardius/go-api-boilerplate/pkg/http/recovery"
	http_response "github.com/vardius/go-api-boilerplate/pkg/http/response"
	log "github.com/vardius/go-api-boilerplate/pkg/log"
	gorouter "github.com/vardius/gorouter/v4"
)

// New provides new router
func New(logger *log.Logger) gorouter.Router {
	http_recovery.WithLogger(logger)
	http_response.WithLogger(logger)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		http_cors.Default().Handler,
		http_response.WithXSS,
		http_response.WithHSTS,
		http_response.AsJSON,
		http_recovery.WithRecover,
	)

	return router
}
