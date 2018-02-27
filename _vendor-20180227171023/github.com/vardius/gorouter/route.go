package gorouter

import (
	"net/http"
)

type route struct {
	middleware middleware
	handler    http.Handler
}

func (r *route) chain() http.Handler {
	if r.handler != nil {
		return r.middleware.handle(r.handler)
	}
	return nil
}

func (r *route) appendMiddleware(m middleware) {
	r.middleware = r.middleware.merge(m)
}

func (r *route) prependMiddleware(m middleware) {
	r.middleware = m.merge(r.middleware)
}

func newRoute(h http.Handler) *route {
	return &route{
		handler:    h,
		middleware: newMiddleware(),
	}
}
