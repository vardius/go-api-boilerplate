package errors_test

import (
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
)

func ExampleNew() {
	err := errors.New("internal error", errors.INTERNAL)

	fmt.Printf("%s\n", errors.ErrorMessage(err))

	// Output:
	// internal error
}

func ExampleWrap() {
	subErr := errors.New("invalid error", errors.INVALID)
	err := errors.Wrap(subErr, "internal error", errors.INTERNAL)

	fmt.Printf("%s\n", errors.ErrorMessage(err))

	// Output:
	// internal error
}
