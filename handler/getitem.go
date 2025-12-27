package handler

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func GetItem(ctx context.Context, s *state, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	key, err := s.itemKey(ctx, *input.TableName, input.Key)
	if err != nil {
		return nil, err
	}
	jsonValue, err := s.kv.Get(ctx, []byte(key))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &dynamodb.GetItemOutput{}, nil
		}
		return nil, err
	}
	var value map[string]types.AttributeValue
	if err := json.Unmarshal(jsonValue, &value, jsonOpts()); err != nil {
		return nil, err
	}
	return &dynamodb.GetItemOutput{
		Item: value,
	}, nil
}
