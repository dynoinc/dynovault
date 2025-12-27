package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-json-experiment/json"
	"github.com/lithammer/shortuuid/v4"
)

func CreateTable(ctx context.Context, s *state, input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error) {
	// TODO: Check if table already exists

	now := time.Now()
	key := fmt.Sprintf("$table:%s", *input.TableName)

	td := &types.TableDescription{
		TableId:              aws.String(shortuuid.New()),
		TableName:            input.TableName,
		TableStatus:          types.TableStatusActive,
		CreationDateTime:     &now,
		AttributeDefinitions: input.AttributeDefinitions,
		KeySchema:            input.KeySchema,
		TableClassSummary: &types.TableClassSummary{
			LastUpdateDateTime: &now,
			TableClass:         types.TableClassStandard,
		},
	}

	jsonValue, err := json.Marshal(td, jsonOpts())
	if err != nil {
		return nil, err
	}

	if err := s.kv.Put(ctx, []byte(key), jsonValue); err != nil {
		return nil, err
	}

	return &dynamodb.CreateTableOutput{
		TableDescription: td,
	}, nil
}
