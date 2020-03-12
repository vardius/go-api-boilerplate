package domain_test

import (
	"fmt"
	"time"

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

func ExampleNewEvent() {
	event, _ := domain.NewEvent(
		uuid.New(),
		"streamName",
		0,
		Test{1, []string{"apple", "peach"}},
	)

	fmt.Printf("%v\n", event.Metadata.StreamName)
	fmt.Printf("%v\n", event.Metadata.StreamVersion)
	fmt.Printf("%s\n", event.Payload)

	// Output:
	// streamName
	// 0
	// {"page":1,"fruits":["apple","peach"]}
}

func ExampleMakeEvent() {
	event, _ := domain.MakeEvent(
		domain.EventMetaData{
			Type:          "type",
			StreamID:      uuid.New(),
			StreamName:    "streamName",
			StreamVersion: 0,
			OccurredAt:    time.Now(),
		},
		[]byte(`{"page":1,"fruits":["apple","peach"]}`),
	)

	fmt.Printf("%v\n", event.Metadata.StreamName)
	fmt.Printf("%v\n", event.Metadata.StreamVersion)
	fmt.Printf("%s\n", event.Payload)

	// Output:
	// streamName
	// 0
	// {"page":1,"fruits":["apple","peach"]}
}
