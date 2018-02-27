package gorouter

import (
	"net/http"
	"os"
)

// HTTP methods constants
const (
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	DELETE  = "DELETE"
	PATCH   = "PATCH"
	OPTIONS = "OPTIONS"
	HEAD    = "HEAD"
	CONNECT = "CONNECT"
	TRACE   = "TRACE"
)

// Router is a micro framwework, HTTP request router, multiplexer, mux
type Router interface {
	// POST adds http.Handler as router handler
	// under POST method and given patter
	POST(pattern string, handler http.Handler)

	// GET adds http.Handler as router handler
	// under GET method and given patter
	GET(pattern string, handler http.Handler)

	// PUT adds http.Handler as router handler
	// under PUT method and given patter
	PUT(pattern string, handler http.Handler)

	// DELETE adds http.Handler as router handler
	// under DELETE method and given patter
	DELETE(pattern string, handler http.Handler)

	// PATCH adds http.Handler as router handler
	// under PATCH method and given patter
	PATCH(pattern string, handler http.Handler)

	// OPTIONS adds http.Handler as router handler
	// under OPTIONS method and given patter
	OPTIONS(pattern string, handler http.Handler)

	// HEAD adds http.Handler as router handler
	// under HEAD method and given patter
	HEAD(pattern string, handler http.Handler)

	// CONNECT adds http.Handler as router handler
	// under CONNECT method and given patter
	CONNECT(pattern string, handler http.Handler)

	// TRACE adds http.Handler as router handler
	// under TRACE method and given patter
	TRACE(pattern string, handler http.Handler)

	// USE adds middleware functions ([]MiddlewareFunc)
	// to whole router branch under given method and patter
	USE(method, pattern string, fs ...MiddlewareFunc)

	// Handle adds http.Handler as router handler
	// under given method and patter
	Handle(method, pattern string, handler http.Handler)

	// HandleFunc adds http.HandlerFunc as router handler
	// under given method and patter
	HandleFunc(method, pattern string, handler http.HandlerFunc)

	// Mount another router instance as a sub tree
	Mount(pattern string, s Router)

	// ServeHTTP dispatches the request to the route handler
	// whose pattern matches the request URL
	ServeHTTP(http.ResponseWriter, *http.Request)

	// ServeFile replies to the request with the
	// contents of the named file or directory.
	ServeFiles(path string, strip bool)

	// NotFound replies to the request with the
	// 404 Error code
	NotFound(http.Handler)

	// NotFound replies to the request with the
	// 405 Error code
	NotAllowed(http.Handler)
}

type router struct {
	routes     *tree
	middleware middleware
	fileServer http.Handler
	notFound   http.Handler
	notAllowed http.Handler
}

func (r *router) POST(p string, f http.Handler) {
	r.Handle(POST, p, f)
}

func (r *router) GET(p string, f http.Handler) {
	r.Handle(GET, p, f)
}

func (r *router) PUT(p string, f http.Handler) {
	r.Handle(PUT, p, f)
}

func (r *router) DELETE(p string, f http.Handler) {
	r.Handle(DELETE, p, f)
}

func (r *router) PATCH(p string, f http.Handler) {
	r.Handle(PATCH, p, f)
}

func (r *router) OPTIONS(p string, f http.Handler) {
	r.Handle(OPTIONS, p, f)
}

func (r *router) HEAD(p string, f http.Handler) {
	r.Handle(HEAD, p, f)
}

func (r *router) CONNECT(p string, f http.Handler) {
	r.Handle(CONNECT, p, f)
}

func (r *router) TRACE(p string, f http.Handler) {
	r.Handle(TRACE, p, f)
}

func (r *router) USE(method, p string, fs ...MiddlewareFunc) {
	r.addMiddleware(method, p, fs...)
}

func (r *router) Handle(m, p string, h http.Handler) {
	r.addRoute(m, p, h)
}

func (r *router) HandleFunc(m, p string, f http.HandlerFunc) {
	r.addRoute(m, p, http.HandlerFunc(f))
}

func (r *router) Mount(p string, subRouter Router) {
	sr, ok := subRouter.(*router)
	if !ok {
		panic("Unable to assert Router")
	}

	r.mergeRouter(p, sr)
}

func (r *router) NotFound(notFound http.Handler) {
	r.notFound = notFound
}

func (r *router) NotAllowed(notAllowed http.Handler) {
	r.notAllowed = notAllowed
}

func (r *router) ServeFiles(path string, strip bool) {
	if path == "" {
		panic("goapi.ServeFiles: empty path!")
	}
	handler := http.FileServer(http.Dir(path))
	if strip {
		handler = http.StripPrefix("/"+path+"/", handler)
	}
	r.fileServer = handler
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	root := r.routes.byID(req.Method)
	if root != nil {
		node, params := root.childByPath(req.URL.Path)
		if node != nil && node.route != nil {
			if h := node.route.chain(); h != nil {
				req = req.WithContext(newContext(req, params))
				h.ServeHTTP(w, req)
				return
			}
		}
	}

	// Handle OPTIONS
	if req.Method == OPTIONS {
		if allow := r.allowed(req.Method, req.URL.Path); len(allow) > 0 {
			w.Header().Set("Allow", allow)
			return
		}
	} else if req.Method == GET && r.fileServer != nil {
		// Handle file serve
		r.serveFiles(w, req)
		return
	} else {
		// Handle 405
		if allow := r.allowed(req.Method, req.URL.Path); len(allow) > 0 {
			w.Header().Set("Allow", allow)
			r.serveNotAllowed(w, req)
			return
		}
	}

	// Handle 404
	r.serveNotFound(w, req)
}

func (r *router) serveNotFound(w http.ResponseWriter, req *http.Request) {
	if r.notFound != nil {
		r.notFound.ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
}

func (r *router) serveNotAllowed(w http.ResponseWriter, req *http.Request) {
	if r.notAllowed != nil {
		r.notAllowed.ServeHTTP(w, req)
	} else {
		http.Error(w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
	}
}

func (r *router) serveFiles(w http.ResponseWriter, req *http.Request) {
	fp := req.URL.Path
	// Return a 404 if the file doesn't exist
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			r.serveNotFound(w, req)
			return
		}
	}
	// Return a 404 if the request is for a directory
	if info.IsDir() {
		r.serveNotFound(w, req)
		return
	}
	r.fileServer.ServeHTTP(w, req)
}

func (r *router) addRoute(method, path string, f http.Handler) {
	root := r.routes.byID(method)
	if root == nil {
		root = newRoot(method)
		r.routes.insert(root)
	}

	paths := splitPath(path)
	route := newRoute(f)
	route.prependMiddleware(r.middleware)

	n := root.addChild(paths)
	n.setRoute(route)
}

func (r *router) addMiddleware(method, path string, fs ...MiddlewareFunc) {
	type recFunc func(recFunc, *node, middleware)
	c := func(c recFunc, n *node, m middleware) {
		if n.route != nil {
			n.route.appendMiddleware(m)
		}
		for _, child := range n.children.statics {
			c(c, child, m)
		}
		for _, child := range n.children.regexps {
			c(c, child, m)
		}
		if n.children.wildcard != nil {
			c(c, n.children.wildcard, m)
		}
	}

	paths := splitPath(path)

	// routes tree roots should be http method nodes only
	for _, root := range r.routes.statics {
		if method == "" || method == root.id {
			node, _ := root.child(paths)
			if node != nil {
				c(c, node, fs)
			}
		}
	}
}

func (r *router) mergeRouter(path string, sr *router) {
	type recFunc func(recFunc, *node, uint8, middleware)
	c := func(c recFunc, n *node, rp uint8, m middleware) {
		n.params += rp
		if n.route != nil {
			n.route.prependMiddleware(m)
		}
		for _, child := range n.children.statics {
			c(c, child, rp, m)
		}
		for _, child := range n.children.regexps {
			c(c, child, rp, m)
		}
		if n.children.wildcard != nil {
			c(c, n.children.wildcard, rp, m)
		}
	}

	// routes tree roots should be http method nodes only
	for _, root := range sr.routes.statics {
		// create new root nodes if we do not have them
		newRootNode := r.routes.byID(root.id)
		if newRootNode == nil {
			newRootNode = newRoot(root.id)
			r.routes.insert(newRootNode)
		}

		paths := splitPath(path)
		n := newRootNode.addChild(paths)
		n.setRoute(root.route)
		n.setChildren(root.children)

		rp := n.params
		n.params = 0

		// prepend global middleware to sub router and increase params if needed
		c(c, n, rp, r.middleware)
	}
}

func (r *router) allowed(method, path string) (allow string) {
	if path == "*" {
		// routes tree roots should be http method nodes only
		for _, root := range r.routes.statics {
			if root.id == OPTIONS {
				continue
			}
			if len(allow) == 0 {
				allow = root.id
			} else {
				allow += ", " + root.id
			}
		}
	} else {
		// routes tree roots should be http method nodes only
		for _, root := range r.routes.statics {
			if root.id == method || root.id == OPTIONS {
				continue
			}

			n, _ := root.childByPath(path)
			if n != nil && n.route != nil {
				if len(allow) == 0 {
					allow = root.id
				} else {
					allow += ", " + root.id
				}
			}
		}
	}
	if len(allow) > 0 {
		allow += ", OPTIONS"
	}
	return allow
}

// New creates new Router instance, return pointer
func New(fs ...MiddlewareFunc) Router {
	return &router{
		routes:     newTree(),
		middleware: newMiddleware(fs...),
	}
}
