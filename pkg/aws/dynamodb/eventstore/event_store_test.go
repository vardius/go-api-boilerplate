package eventstore

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestNew(t *testing.T) {
	bus := New("test", &aws.Config{Region: aws.String("us-west-2")})

	if bus == nil {
		t.Fail()
	}
}
