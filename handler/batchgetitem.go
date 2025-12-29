package handler

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func BatchGetItem(
	ctx context.Context,
	s *state,
	input *dynamodb.BatchGetItemInput,
) (*dynamodb.BatchGetItemOutput, error) {
	responses := map[string][]map[string]types.AttributeValue{}

	for tableName, requestItem := range input.RequestItems {
		for _, attr := range requestItem.Keys {
			key, err := s.itemKey(ctx, tableName, attr)
			if err != nil {
				return nil, err
			}
			result, err := s.kv.Get(ctx, []byte(key))
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					continue
				}
				return nil, err
			}
			var value map[string]types.AttributeValue
			if err := json.Unmarshal(result.Value, &value, jsonOpts()); err != nil {
				return nil, err
			}
			responses[tableName] = append(responses[tableName], value)
		}
	}
	return &dynamodb.BatchGetItemOutput{
		Responses: responses,
	}, nil
}
