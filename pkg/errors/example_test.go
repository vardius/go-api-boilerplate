package errors_test

import (
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func ExampleNew() {
	err := errors.New("example")

	fmt.Printf("%s\n", err)

	// Output:
	// example:
	// 	/home/travis/gopath/src/github.com/vardius/go-api-boilerplate/pkg/errors/example_test.go:10
}

func ExampleWrap() {
	subErr := errors.New("example")
	err := errors.Wrap(subErr)

	fmt.Printf("%s\n", err)

	// Output:
	// /home/travis/gopath/src/github.com/vardius/go-api-boilerplate/pkg/errors/example_test.go:20
	// example:
	// 	/home/travis/gopath/src/github.com/vardius/go-api-boilerplate/pkg/errors/example_test.go:19
}
