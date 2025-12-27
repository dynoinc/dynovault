package handler

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func DeleteItem(ctx context.Context, s *state, input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	key, err := s.itemKey(ctx, *input.TableName, input.Key)
	if err != nil {
		return nil, err
	}
	if err := s.kv.Delete(ctx, []byte(key)); err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return &dynamodb.DeleteItemOutput{}, nil
}
