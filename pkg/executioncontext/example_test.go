package executioncontext_test

import (
	"context"
	"fmt"
	"sort"

	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

func ExampleContextWithFlag() {
	ctx := context.Background()
	ctx = executioncontext.ContextWithFlag(ctx, "test")

	fmt.Printf("%v\n", executioncontext.HasFlag(ctx, "test"))

	// Output:
	// true
}

func ExampleHasFlag() {
	ctx := context.Background()

	fmt.Printf("%v\n", executioncontext.HasFlag(ctx, "test"))

	ctx = executioncontext.ContextWithFlag(ctx, "test")

	fmt.Printf("%v\n", executioncontext.HasFlag(ctx, "test"))

	// Output:
	// false
	// true
}

func ExampleFlagsFromContext() {
	ctx := context.Background()
	flags := executioncontext.FlagsFromContext(ctx)

	fmt.Printf("%v\n", flags)

	ctx = executioncontext.ContextWithFlag(ctx, "foo")
	ctx = executioncontext.ContextWithFlag(ctx, "bar")
	flags = executioncontext.FlagsFromContext(ctx)

	sort.Strings(flags)
	fmt.Printf("%v\n", flags)

	// Output:
	// []
	// [bar foo]
}
