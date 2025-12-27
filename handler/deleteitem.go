package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func DeleteItem(ctx context.Context, s *state, input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	key := *input.TableName
	for k, v := range input.Key {
		if sv, ok := v.(*types.AttributeValueMemberS); ok {
			key = fmt.Sprintf("%s:%s-%s", key, k, sv.Value)
		}
	}
	if err := s.kv.Delete(ctx, []byte(key)); err != nil {
		return nil, err
	}
	return &dynamodb.DeleteItemOutput{}, nil
}
