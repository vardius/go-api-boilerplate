package errors_test

import (
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func ExampleNew() {
	err := errors.New("example")

	fmt.Printf("%s\n", err)

	// Output:
	// example
}

func ExampleWrap() {
	subErr := errors.New("example")
	err := errors.Wrap(subErr)

	fmt.Printf("%s\n", err)

	// Output:
	// example
}
