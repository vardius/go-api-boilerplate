package dynamodb

import (
	"app/pkg/domain"
	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type eventStore struct {
	service *dynamodb.DynamoDB
	tableName string
}

func (s *eventStore) Store(events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	//todo: check event version
	for _, e := range events {
		item, err := dynamodbattribute.MarshalMap(e)
		if err != nil {
			return err
		}
		putParams := &dynamodb.PutItemInput{
			TableName:           aws.String(s.tableName),
			ConditionExpression: aws.String("
				attribute_not_exists(id) AND
				attribute_not_exists(metadata) AND
				attribute_not_exists(payload)
			"),
			Item:                item,
		}
		if _, err = s.service.PutItem(putParams); err != nil {
			if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ConditionalCheckFailedException" {
				return err
			}
			return err
		}
	}

	return nil
}

func (s *eventStore) Get(id uuid.UUID) (*domain.Event, error) {
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
		return es[0], err
	}

	return nil, err
}

func (s *eventStore) FindAll() []*domain.Event {
	params := &dynamodb.QueryInput{
		TableName: aws.String(s.tableName),
		ConsistentRead: aws.Bool(true),
	}
	
	es, _ := s.query(params)
	
	if es == nil {
		es = make([]*domain.Event, 0)
	}

	return es
}

func (s *eventStore) GetStream(streamId uuid.UUID, streamName string) []*domain.Event {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("metadata.streamId = :streamId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":streamId": {S: aws.String(streamId.String())},
		},
		ConsistentRead: aws.Bool(true),
	}
	
	es, _ := s.query(params)
	
	if es == nil {
		es = make([]*domain.Event, 0)
	}

	return es
}

func (s *eventStore) query(params &dynamodb.QueryInput) (*domain.Event, error) {
	resp, err := s.service.Query(params)
	if err != nil {
		return nil, err
	}

	if len(resp.Items) == 0 {
		return nil, ErrEventNotFound
	}

	es := make([]*domain.Event, len(resp.Items))
	for i, item := range resp.Items {
		e := domain.Event{}
		if err := dynamodbattribute.UnmarshalMap(item, &e); err != nil {
			return nil, err
		}
		es[i] = e
	}

	return es, nil
}

func NewEventStore(string tableName, config *aws.Config) domain.EventStore {
	if tableName == "" {
		tableName = "events"
	}

	return &eventStore{
		service: dynamodb.New(session.New(), awsConfig),
		tableName:  tableName,
	}
}
