package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
)

func DescribeTable(ctx context.Context, s *state, input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	key := fmt.Sprintf("$table:%s", *input.TableName)
	result, err := s.kv.Get(ctx, []byte(key))
	if err != nil {
		return nil, err
	}

	var td types.TableDescription
	if err := json.Unmarshal(result.Value, &td, jsonOpts()); err != nil {
		return nil, err
	}

	return &dynamodb.DescribeTableOutput{
		Table: &td,
	}, nil
}
