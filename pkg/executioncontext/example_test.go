package executioncontext_test

import (
	"context"
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

func ExampleWithFlag() {
	ctx := context.Background()
	ctx = executioncontext.WithFlag(ctx, executioncontext.LIVE)

	fmt.Printf("%v\n", executioncontext.Has(ctx, executioncontext.LIVE))

	// Output:
	// true
}

func ExampleHas() {
	ctx := context.Background()

	fmt.Printf("%v\n", executioncontext.Has(ctx, executioncontext.LIVE))

	ctx = executioncontext.WithFlag(ctx, executioncontext.LIVE)

	fmt.Printf("%v\n", executioncontext.Has(ctx, executioncontext.LIVE))

	// Output:
	// false
	// true
}

func ExampleFromContext() {
	ctx := context.Background()
	flags := executioncontext.FromContext(ctx)

	fmt.Printf("%v\n", flags)

	ctx = executioncontext.WithFlag(ctx, executioncontext.LIVE)
	ctx = executioncontext.WithFlag(ctx, executioncontext.REPLAY)
	flags = executioncontext.FromContext(ctx)

	fmt.Printf("%v\n", flags)

	// Output:
	// 0
	// 3
}
