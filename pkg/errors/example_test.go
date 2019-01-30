package errors_test

import (
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func ExampleNew() {
	err := errors.New(errors.INTERNAL, "internal error")

	fmt.Printf("%s\n", errors.ErrorMessage(err))

	// Output:
	// internal error
}

func ExampleWrap() {
	subErr := errors.New(errors.INVALID, "invalid error")
	err := errors.Wrap(subErr, errors.INTERNAL, "internal error")

	fmt.Printf("%s\n", errors.ErrorMessage(err))

	// Output:
	// internal error
}

func ExampleNewf() {
	err := errors.Newf(errors.INTERNAL, "%s %s", "internal", "error")

	fmt.Printf("%s\n", errors.ErrorMessage(err))

	// Output:
	// internal error
}

func ExampleWrapf() {
	subErr := errors.New(errors.INVALID, "invalid error")
	err := errors.Wrapf(subErr, errors.INTERNAL, "%s %s", "internal", "error")

	fmt.Printf("%s\n", errors.ErrorMessage(err))

	// Output:
	// internal error
}
