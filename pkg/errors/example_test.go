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
	// example:
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:12
}

func ExampleWrap() {
	subErr := apperrors.New("example")
	err := apperrors.Wrap(subErr)

	fmt.Printf("%s", err)

	// Output:
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:23
	// example:
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:22
}

func ExampleWrap_second() {
	subErr := apperrors.Wrap(fmt.Errorf("test"))
	err := apperrors.Wrap(subErr)

	deeper := apperrors.Wrap(fmt.Errorf("test: %w", err))
	all := apperrors.Wrap(deeper)

	fmt.Printf("%s", all)

	// Output:
	// /home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:38
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:37
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:35
	// test:
	// 	/home/runner/work/go-api-boilerplate/go-api-boilerplate/pkg/errors/example_test.go:34
}
