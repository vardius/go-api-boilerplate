package main

import (
	"net/http"

	http_cors "github.com/rs/cors"
	http_middleware "github.com/vardius/go-api-boilerplate/internal/http/middleware"
	http_metadata_middleware "github.com/vardius/go-api-boilerplate/internal/http/middleware/metadata"
	log "github.com/vardius/go-api-boilerplate/internal/log"
	gorouter "github.com/vardius/gorouter/v4"
)

// NewRouter provides new router
func NewRouter(logger *log.Logger) gorouter.Router {
	// Global middleware
	router := gorouter.New(
		http_middleware.Recover(logger),
		http_metadata_middleware.WithMetadata(),
		http_middleware.Logger(logger),
		http_cors.Default().Handler,
		http_middleware.XSS(),
		http_middleware.HSTS(),
		http_middleware.Metrics(),
		http_middleware.LimitRequestBody(int64(10<<20)), // 10 MB is a lot of text.
	)

	// Liveness probes are to indicate that your application is running
	router.GET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello"))
	}))
	router.GET("/v1/health", BuildLivenessHandler())
	router.GET("/v1/readiness", BuildLivenessHandler())

	return router
}

// BuildLivenessHandler provides liveness handler
func BuildLivenessHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}

	return http.HandlerFunc(fn)
}
