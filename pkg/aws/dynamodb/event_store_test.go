package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestNewEventStore(t *testing.T) {
	bus := NewEventStore("test", &aws.Config{Region: aws.String("us-west-2")})

	if bus == nil {
		t.Fail()
	}
}
