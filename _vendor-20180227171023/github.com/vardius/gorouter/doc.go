/*
Package gorouter provide request router with middleware

Router

The router determines how to handle that request.
GoRouter uses a routing tree. Once one branch of the tree matches, only routes inside that branch are considered,
not any routes after that branch. When instantiating router, the root node of tree is created.

Route types

- Static `/hello` (will match requests matching given route)

- Named `/{name}` (will match requests matching given route scheme)

- Regexp `/{name:[a-z]+}` (will match requests matching given route scheme and its regexp)

Wildcards

The values of *named parameter* or *regexp parameters* are accessible via *request context*
`params, ok := gorouter.FromContext(req.Context())`.
You can get the value of a parameter either by its index in the slice, or by using the `params.Value(name)` method:
`:name` or `/{name:[a-z]+}` can be retrieved by `params.Value("name")`.

Defining Routes

A full route definition contain up to three parts:

1. HTTP method under which route will be available

2. The URL path route. This is matched against the URL passed to the router,
and can contain named wildcard placeholders *(e.g. {placeholder})* to match dynamic parts in the URL.

3. `http.HandleFunc`, which tells the router to handle matched requests to the router with handler.

Take the following example:

	router.GET("/hello/{name:r([a-z]+)go}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := gorouter.FromContext(r.Context())
		fmt.Fprintf(w, "hello, %s!\n", params.Value("name"))
	}))

In this case, the route is matched by `/hello/rxxxxxgo` for example,
because the `:name` wildcard matches the regular expression wildcard given (`r([a-z]+)go`). However,
`/hello/foo` does not match, because "foo" fails the *name* wildcard. When using wildcards,
these are returned in the map from request context. The part of the path that the wildcard matched (e.g. *rxxxxxgo*) is used as value.

Basic example:

	package main

	import (
		"fmt"
		"log"
		"net/http"

		"github.com/vardius/gorouter"
	)

	func Index(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome!\n")
	}

	func Hello(w http.ResponseWriter, r *http.Request) {
		params, _ := gorouter.FromContext(r.Context())
		fmt.Fprintf(w, "hello, %s!\n", params.Value("name"))
	}

	func main() {
		router := gorouter.New()
		router.GET("/", http.HandlerFunc(Index))
		router.GET("/hello/{name}", http.HandlerFunc(Hello))

		log.Fatal(http.ListenAndServe(":8080", router))
	}
*/
package gorouter
