package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func DeleteTable(ctx context.Context, s *state, input *dynamodb.DeleteTableInput) (*dynamodb.DeleteTableOutput, error) {
	key := fmt.Sprintf("$table:%s", *input.TableName)
	jsonValue, err := s.kv.Get(ctx, []byte(key))
	if err != nil {
		return nil, err
	}

	var td types.TableDescription
	if err = json.Unmarshal(jsonValue, &td, jsonOpts()); err != nil {
		return nil, err
	}

	if err = s.kv.Delete(ctx, []byte(key)); err != nil {
		return nil, err
	}

	s.keySchema.Delete(*input.TableName)

	return &dynamodb.DeleteTableOutput{
		TableDescription: &td,
	}, nil
}
