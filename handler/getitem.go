package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func GetItem(ctx context.Context, s *state, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	key := *input.TableName
	for k, v := range input.Key {
		if sv, ok := v.(*types.AttributeValueMemberS); ok {
			key = fmt.Sprintf("%s:%s-%s", key, k, sv.Value)
		}
	}
	jsonValue, err := s.kv.Get(ctx, []byte(key))
	if err != nil {
		return &dynamodb.GetItemOutput{}, nil
	}
	var value map[string]types.AttributeValue
	if err := json.Unmarshal(jsonValue, &value, jsonOpts()); err != nil {
		return nil, err
	}
	return &dynamodb.GetItemOutput{
		Item: value,
	}, nil
}
