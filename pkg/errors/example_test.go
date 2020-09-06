package errors_test

import (
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func ExampleNew() {
	err := errors.New("example")

	fmt.Printf("%s", err)

	// Output:
	// example:
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:10
}

func ExampleWrap() {
	subErr := errors.New("example")
	err := errors.Wrap(subErr)

	fmt.Printf("%s", err)

	// Output:
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:21
	// example:
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:20
}
