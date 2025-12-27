package handler

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func BatchWriteItem(
	ctx context.Context,
	s *state,
	input *dynamodb.BatchWriteItemInput,
) (*dynamodb.BatchWriteItemOutput, error) {
	for tableName, writeRequests := range input.RequestItems {
		for _, writeRequest := range writeRequests {
			// A write request can contain delete XOR put
			// the AWS SDK should validate that for us
			if writeRequest.DeleteRequest != nil {
				key, err := s.itemKey(ctx, tableName, writeRequest.DeleteRequest.Key)
				if err != nil {
					return nil, err
				}
				if err := s.kv.Delete(ctx, []byte(key)); err != nil && !errors.Is(err, ErrNotFound) {
					return nil, err
				}
			} else {
				key, err := s.itemKey(ctx, tableName, writeRequest.PutRequest.Item)
				if err != nil {
					return nil, err
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
