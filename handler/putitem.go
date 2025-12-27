package handler

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-json-experiment/json"
)

func PutItem(ctx context.Context, s *state, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	key, err := s.itemKey(ctx, *input.TableName, input.Item)
	if err != nil {
		return nil, err
	}
	// Process the put
	// PutRequest.Item is map[string]types.AttributeValue
	jsonValue, err := json.Marshal(input.Item, jsonOpts())
	if err != nil {
		return nil, err
	}

	if err := s.kv.Put(ctx, []byte(key), jsonValue); err != nil {
		return nil, err
	}
	return &dynamodb.PutItemOutput{}, nil
}
