// +build docker

package errors_test

import (
	"fmt"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func ExampleNew() {
	err := apperrors.New("example")

	fmt.Printf("%s", err)

	// Output:
	// example
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:12
}

func ExampleWrap() {
	subErr := apperrors.New("example")
	err := apperrors.Wrap(subErr)

	fmt.Printf("%s", err)

	// Output:
	// example
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:22
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:23
}

func ExampleWrap_second() {
	originalErr := fmt.Errorf("original")
	wrappedErr := apperrors.Wrap(originalErr)

	err := fmt.Errorf("test2: %w", wrappedErr)

	fmt.Printf("%s", err)

	// Output:
	// test2: original
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:35
}

func ExampleWrap_third() {
	first := apperrors.Wrap(fmt.Errorf("first"))
	wrapped := apperrors.Wrap(first)

	second := apperrors.Wrap(fmt.Errorf("second: %w", wrapped))
	third := apperrors.Wrap(fmt.Errorf("third: %w", second))
	err := apperrors.Wrap(third)

	fmt.Printf("%s\n\n", err)

	// Output:
	// third: second: first
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:47
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:48
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:50
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:51
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:52
}
