package itest

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	db *dynamodb.Client
}

func New(t *testing.T, url string) *Suite {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("ID", "SECRET_KEY", "TOKEN"),
		),
	)
	require.NoError(t, err)

	db := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(url)
	})
	return &Suite{db: db}
}

func (s *Suite) TestCreateTable() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	_, err = s.db.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
		TableName: aws.String(testTableName),
	})
	require.NoError(s.T(), err)
}

func (s *Suite) TestInvalidCreateTable() {
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(""),
	})
	require.Error(s.T(), err)
}

func (s *Suite) TestDeleteTable() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	response, err := s.db.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(testTableName),
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), response.TableDescription)
	require.Equal(s.T(), *response.TableDescription.TableName, testTableName)

	_, err = s.db.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
		TableName: aws.String(testTableName),
	})
	require.Error(s.T(), err)
}

func (s *Suite) TestPutItem() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	_, err = s.db.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(testTableName),
		Item: map[string]types.AttributeValue{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"value": &types.AttributeValueMemberS{Value: "Test Value"},
		},
	})
	require.NoError(s.T(), err)
}

func (s *Suite) TestGetItem() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	_, err = s.db.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(testTableName),
		Item: map[string]types.AttributeValue{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"value": &types.AttributeValueMemberS{Value: "Test Value"},
		},
	})

	require.NoError(s.T(), err)
	response, err := s.db.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(testTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "1"},
		},
	})
	require.NotEmpty(s.T(), response.Item)
	require.EqualValues(s.T(), response.Item["id"].(*types.AttributeValueMemberS).Value, "1")
	require.EqualValues(s.T(), response.Item["value"].(*types.AttributeValueMemberS).Value, "Test Value")
	require.NoError(s.T(), err)
}

func (s *Suite) TestDeleteItem() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	_, err = s.db.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(testTableName),
		Item: map[string]types.AttributeValue{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"value": &types.AttributeValueMemberS{Value: "Test Value"},
		},
	})
	require.NoError(s.T(), err)

	response, err := s.db.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(testTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "1"},
		},
	})
	require.NotEmpty(s.T(), response.Item)
	require.EqualValues(s.T(), response.Item["id"].(*types.AttributeValueMemberS).Value, "1")
	require.EqualValues(s.T(), response.Item["value"].(*types.AttributeValueMemberS).Value, "Test Value")
	require.NoError(s.T(), err)

	_, err = s.db.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String("TestTable"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "1"},
		},
	})
	require.NoError(s.T(), err)

	response, err = s.db.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(testTableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "1"},
		},
	})
	require.Empty(s.T(), response.Item)
	require.NoError(s.T(), err)
}

func (s *Suite) TestBatchWrite() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})

	require.NoError(s.T(), err)
	_, err = s.db.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			testTableName: {
				{
					PutRequest: &types.PutRequest{
						Item: map[string]types.AttributeValue{
							"id":    &types.AttributeValueMemberS{Value: "1"},
							"value": &types.AttributeValueMemberS{Value: "test value"},
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)
}

func (s *Suite) TestBatchGet() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	testValue := "Test Value"
	_, err = s.db.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			testTableName: {
				{
					PutRequest: &types.PutRequest{
						Item: map[string]types.AttributeValue{
							"id":    &types.AttributeValueMemberS{Value: "1"},
							"value": &types.AttributeValueMemberS{Value: testValue},
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)

	response, err := s.db.BatchGetItem(context.Background(), &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			testTableName: {
				ProjectionExpression: aws.String("id"),
				Keys: []map[string]types.AttributeValue{
					{
						"id": &types.AttributeValueMemberS{Value: "1"},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), response.Responses)

	responseItems := response.Responses[testTableName]
	require.NotEmpty(s.T(), responseItems)
	var testItem map[string]types.AttributeValue
	for _, item := range responseItems {
		if item["id"].(*types.AttributeValueMemberS).Value == "1" {
			testItem = item
		}
	}
	require.NotNil(s.T(), testItem)
	require.EqualValues(s.T(), testItem["value"].(*types.AttributeValueMemberS).Value, testValue)
}

func (s *Suite) TestBatchDelete() {
	testTableName := "TestTable"
	_, err := s.db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("value"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	require.NoError(s.T(), err)

	_, err = s.db.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			testTableName: {
				{
					PutRequest: &types.PutRequest{
						Item: map[string]types.AttributeValue{
							"id": &types.AttributeValueMemberS{Value: "1"},
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)

	_, err = s.db.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			testTableName: {
				{
					DeleteRequest: &types.DeleteRequest{
						Key: map[string]types.AttributeValue{
							"id": &types.AttributeValueMemberS{Value: "1"},
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)

	response, err := s.db.BatchGetItem(context.Background(), &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			testTableName: {
				ProjectionExpression: aws.String("id"),
				Keys: []map[string]types.AttributeValue{
					{
						"id": &types.AttributeValueMemberS{Value: "1"},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), response.Responses)
}
