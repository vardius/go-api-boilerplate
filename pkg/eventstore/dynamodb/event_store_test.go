package eventstore

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestNew(t *testing.T) {
	store := New("test", &aws.Config{Region: aws.String("us-west-2")})

	if store == nil {
		t.Fail()
	}
}
