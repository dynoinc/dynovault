package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func getPartitionKey(ctx context.Context, s *state, tableName string) (string, error) {
	key, found := s.partitionKey.Load(tableName)
	if found {
		return key.(string), nil
	}

	// Get the table key schema to determine which keys are pks
	describeTableOutput, err := DescribeTable(ctx, s, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return "", err
	}

	hashKey := ""
	for _, keySchemaElement := range describeTableOutput.Table.KeySchema {
		if keySchemaElement.KeyType == types.KeyTypeHash {
			hashKey = *keySchemaElement.AttributeName
			break
		}
	}
	s.partitionKey.Store(tableName, hashKey)
	return hashKey, nil
}

func BatchWriteItem(
	ctx context.Context,
	s *state,
	input *dynamodb.BatchWriteItemInput,
) (*dynamodb.BatchWriteItemOutput, error) {
	for tableName, writeRequests := range input.RequestItems {
		partitionKey, err := getPartitionKey(ctx, s, tableName)
		if err != nil {
			return nil, err
		}

		for _, writeRequest := range writeRequests {
			key := tableName
			// A write request can contain delete XOR put
			// the AWS SDK should validate that for us
			if writeRequest.DeleteRequest != nil {
				for keyName, keyValue := range writeRequest.DeleteRequest.Key {
					if sv, ok := keyValue.(*types.AttributeValueMemberS); ok {
						key = fmt.Sprintf("%s:%s-%s", key, keyName, sv.Value)
					}
				}
				if err := s.kv.Delete(ctx, []byte(key)); err != nil {
					return nil, err
				}
			} else {
				if sv, ok := writeRequest.PutRequest.Item[partitionKey].(*types.AttributeValueMemberS); ok {
					key = fmt.Sprintf("%s:%s-%s", key, partitionKey, sv.Value)
				}
				// Process the put
				// PutRequest.Item is map[string]types.AttributeValue
				jsonValue, err := json.Marshal(writeRequest.PutRequest.Item, jsonOpts())
				if err != nil {
					return nil, err
				}

				if err := s.kv.Put(ctx, []byte(key), jsonValue); err != nil {
					return nil, err
				}
			}
		}
	}
	return &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]types.WriteRequest{},
	}, nil
}
