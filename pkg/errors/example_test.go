//go:build docker
// +build docker

package errors_test

import (
	"fmt"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func ExampleNew() {
	err := apperrors.New("example")

	fmt.Printf("%s\n", err.Error())
	var e *apperrors.AppError
	if errors.As(err, &e) {
		fmt.Printf("%s", e.StackTrace())
	}

	// Output:
	// example
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:13
}

func ExampleWrap() {
	subErr := apperrors.New("example")
	err := apperrors.Wrap(subErr)

	fmt.Printf("%s\n", err.Error())
	var e *apperrors.AppError
	if errors.As(err, &e) {
		fmt.Printf("%s", e.StackTrace())
	}

	// Output:
	// example
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:28
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:27
}

func ExampleWrap_second() {
	originalErr := fmt.Errorf("original")
	wrappedErr := apperrors.Wrap(originalErr)

	err := fmt.Errorf("test2: %w", wrappedErr)

	fmt.Printf("%s\n", err.Error())
	var e *apperrors.AppError
	if errors.As(err, &e) {
		fmt.Printf("%s", e.StackTrace())
	}

	// Output:
	// test2: original
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:44
}

func ExampleWrap_third() {
	first := apperrors.Wrap(fmt.Errorf("first"))
	wrapped := apperrors.Wrap(first)

	second := apperrors.Wrap(fmt.Errorf("second: %w", wrapped))
	third := apperrors.Wrap(fmt.Errorf("third: %w", second))
	err := apperrors.Wrap(third)

	fmt.Printf("%s\n", err.Error())
	var e *apperrors.AppError
	if errors.As(err, &e) {
		fmt.Printf("%s", e.StackTrace())
	}

	// Output:
	// third: second: first
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:65
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:64
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:63
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:61
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:60
}
