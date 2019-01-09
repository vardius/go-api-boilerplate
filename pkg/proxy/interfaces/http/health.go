package http

import (
	"net/http"

	"github.com/vardius/gorouter"
)

// AddHealthCheckRoutes adds health checks route
func AddHealthCheckRoutes(router gorouter.Router) {
	// Liveness probes are to indicate that your application is running
	router.GET("/healthz", http.HandlerFunc(liveness))
	// Readiness is meant to check if your application is ready to serve traffic
	router.GET("/readiness", http.HandlerFunc(readiness))
}

func liveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}
