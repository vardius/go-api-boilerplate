Vardius - gorouter
================
[![Build Status](https://travis-ci.org/vardius/gorouter.svg?branch=master)](https://travis-ci.org/vardius/gorouter)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/gorouter)](https://goreportcard.com/report/github.com/vardius/gorouter)
[![codecov](https://codecov.io/gh/vardius/gorouter/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/gorouter)
[![](https://godoc.org/github.com/vardius/gorouter?status.svg)](http://godoc.org/github.com/vardius/gorouter)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/gorouter/blob/master/LICENSE.md)
[![Beerpay](https://beerpay.io/vardius/gorouter/badge.svg?style=beer-square)](https://beerpay.io/vardius/gorouter)
[![Beerpay](https://beerpay.io/vardius/gorouter/make-wish.svg?style=flat-square)](https://beerpay.io/vardius/gorouter?focus=wish)

Go Server/API micro framwework, HTTP request router, multiplexer, mux.

ABOUT
==================================================
Contributors:

* [Rafał Lorenz](http://rafallorenz.com)

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/gorouter/issues) to manage them.

HOW TO USE
==================================================

1. [GoDoc](http://godoc.org/github.com/vardius/gorouter)
2. [Documentation](https://github.com/vardius/gorouter/wiki)
3. [Benchmarks](https://github.com/vardius/gorouter/wiki/Benchmarks)
4. [Go Server/API boilerplate using best practices DDD CQRS ES](https://github.com/vardius/go-api-boilerplate)

## Basic example
```go
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
```

License
-------

This package is released under the MIT license. See the complete license in the package:

[LICENSE](LICENSE.md)
