package handler

import (
	"context"
	"fmt"

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
		key := tableName
		for _, attr := range requestItem.Keys {
			// Flatten the keys into one string
			// TODO: ordering may mess us up here
			for k, v := range attr {
				if sv, ok := v.(*types.AttributeValueMemberS); ok {
					key = fmt.Sprintf("%s:%s-%s", key, k, sv.Value)
				}
			}
			jsonValue, err := s.kv.Get(ctx, []byte(key))
			if err != nil {
				//return nil, err
				continue
			}
			var value map[string]types.AttributeValue
			if err := json.Unmarshal(jsonValue, &value, jsonOpts()); err != nil {
				return nil, err
			}
			responses[tableName] = append(responses[tableName], value)
		}
	}
	return &dynamodb.BatchGetItemOutput{
		Responses: responses,
	}, nil
}
