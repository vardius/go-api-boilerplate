/*
Package eventstore provides dynamodb implementation of domain event store
*/
package eventstore

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	baseeventstore "github.com/vardius/go-api-boilerplate/internal/eventstore"
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
			return errors.Wrap(err, errors.INTERNAL, "EventStore events marshal error")
		}
		putParams := &dynamodb.PutItemInput{
			TableName:           aws.String(s.tableName),
			ConditionExpression: aws.String("attribute_not_exists(id) AND attribute_not_exists(metadata) AND attribute_not_exists(payload)"),
			Item:                item,
		}
		if _, err = s.service.PutItem(putParams); err != nil {
			if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ConditionalCheckFailedException" {
				return errors.Wrap(err, errors.INTERNAL, "EventStore PutItem request failureerror")
			}
			return errors.Wrap(err, errors.INTERNAL, "EventStore PutItem error")
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
	if len(es) > 0 {
		return es[0], errors.Wrap(err, errors.INTERNAL, "Query events failed")
	}

	return domain.NullEvent, nil
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
		return nil, errors.Wrap(err, errors.INTERNAL, "Query failed")
	}

	if len(resp.Items) == 0 {
		return nil, errors.Wrap(ErrEventNotFound, errors.NOTFOUND, "Not found any items")
	}

	es := make([]domain.Event, len(resp.Items))
	for i, item := range resp.Items {
		e := domain.Event{}
		if err := dynamodbattribute.UnmarshalMap(item, &e); err != nil {
			return nil, errors.Wrap(err, errors.INTERNAL, "Unmarshal events failed")
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

	return &eventStore{
		service:   dynamodb.New(session.New(), config),
		tableName: tableName,
	}
}
