/*
Package memory provides dynamodb implementation of domain event store
*/
package memory

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	baseeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type eventStore struct {
	service   *dynamodb.DynamoDB
	tableName string
}

func (s *eventStore) Store(events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	// @TODO: check event version
	for _, e := range events {
		item, err := dynamodbattribute.MarshalMap(e)
		if err != nil {
			return fmt.Errorf("could not parse event: %w", err)
		}
		putParams := &dynamodb.PutItemInput{
			TableName:           aws.String(s.tableName),
			ConditionExpression: aws.String("attribute_not_exists(id) AND attribute_not_exists(metadata) AND attribute_not_exists(payload)"),
			Item:                item,
		}
		if _, err = s.service.PutItem(putParams); err != nil {
			if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ConditionalCheckFailedException" {
				return fmt.Errorf("PutItem request failureerror: %w", err)
			}
			return fmt.Errorf("PutItem error: %w", err)
		}
	}

	return nil
}

func (s *eventStore) Get(id uuid.UUID) (domain.Event, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("id = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {S: aws.String(id.String())},
		},
		ConsistentRead: aws.Bool(true),
	}

	es, err := s.query(params)
	if err != nil {
		return es[0], fmt.Errorf("query events failed: %w", err)
	}

	if len(es) > 0 {
		return es[0], nil
	}

	return domain.NullEvent, baseeventstore.ErrEventNotFound
}

func (s *eventStore) FindAll() []domain.Event {
	params := &dynamodb.QueryInput{
		TableName:      aws.String(s.tableName),
		ConsistentRead: aws.Bool(true),
	}

	es, _ := s.query(params)

	if es == nil {
		return make([]domain.Event, 0)
	}

	return es
}

func (s *eventStore) GetStream(streamID uuid.UUID, streamName string) []domain.Event {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("metadata.streamID = :streamID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":streamID": {S: aws.String(streamID.String())},
		},
		ConsistentRead: aws.Bool(true),
	}

	es, _ := s.query(params)

	if es == nil {
		return make([]domain.Event, 0)
	}

	return es
}

func (s *eventStore) query(params *dynamodb.QueryInput) ([]domain.Event, error) {
	resp, err := s.service.Query(params)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if len(resp.Items) == 0 {
		return nil, baseeventstore.ErrEventNotFound
	}

	es := make([]domain.Event, len(resp.Items))
	for i, item := range resp.Items {
		e := domain.Event{}
		if err := dynamodbattribute.UnmarshalMap(item, &e); err != nil {
			return nil, fmt.Errorf("unmarshal events failed: %w", err)
		}
		es[i] = e
	}

	return es, nil
}

// New creates new dynamodb event store
func New(tableName string, config *aws.Config) baseeventstore.EventStore {
	if tableName == "" {
		tableName = "events"
	}

	// @TODO:  handle error
	s, _ := session.NewSession()

	return &eventStore{
		service:   dynamodb.New(s, config),
		tableName: tableName,
	}
}
