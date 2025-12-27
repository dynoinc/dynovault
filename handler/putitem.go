package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func PutItem(ctx context.Context, s *state, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	// Get the table key schema to determine which keys are pks
	describeTableOutput, err := DescribeTable(ctx, s, &dynamodb.DescribeTableInput{
		TableName: input.TableName,
	})
	if err != nil {
		return nil, err
	}
	key := *input.TableName
	for keyName, keyValue := range input.Item {
		for _, keySchemaElement := range describeTableOutput.Table.KeySchema {
			if keyName == *keySchemaElement.AttributeName {
				if sv, ok := keyValue.(*types.AttributeValueMemberS); ok {
					key = fmt.Sprintf("%s:%s-%s", key, keyName, sv.Value)
				}
			}
		}
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
