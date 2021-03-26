package domain_test

import (
	"fmt"
	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

type Test struct {
	Page   int      `json:"page"`
	Fruits []string `json:"fruits"`
}

func (e Test) GetType() string {
	return ""
}

func ExampleNewEventFromRawEvent() {
	event, _ := domain.NewEventFromRawEvent(
		uuid.New(),
		"streamName",
		0,
		Test{1, []string{"apple", "peach"}},
	)

	fmt.Printf("%v\n", event.StreamName)
	fmt.Printf("%v\n", event.StreamVersion)
	fmt.Printf("%v\n", event.Payload)

	// Output:
	// streamName
	// 0
	// {1 [apple peach]}
}
