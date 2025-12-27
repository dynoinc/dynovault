package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (s *state) keyNames(ctx context.Context, tableName string) ([]string, error) {
	if cached, ok := s.keySchema.Load(tableName); ok {
		return cached.([]string), nil
	}

	out, err := DescribeTable(ctx, s, &dynamodb.DescribeTableInput{
		TableName: &tableName,
	})
	if err != nil {
		return nil, err
	}

	names := schemaKeyNames(out.Table.KeySchema)
	s.keySchema.Store(tableName, names)
	return names, nil
}

func schemaKeyNames(schema []types.KeySchemaElement) []string {
	names := make([]string, 0, 2)
	for _, elem := range schema {
		if elem.KeyType == types.KeyTypeHash && elem.AttributeName != nil {
			names = append(names, *elem.AttributeName)
		}
	}
	for _, elem := range schema {
		if elem.KeyType == types.KeyTypeRange && elem.AttributeName != nil {
			names = append(names, *elem.AttributeName)
		}
	}
	return names
}

func (s *state) itemKey(ctx context.Context, tableName string, attrs map[string]types.AttributeValue) (string, error) {
	names, err := s.keyNames(ctx, tableName)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("table %q has no key schema", tableName)
	}

	var b strings.Builder
	b.WriteString(tableName)
	for _, name := range names {
		av, ok := attrs[name]
		if !ok {
			return "", fmt.Errorf("missing key attribute %q", name)
		}
		sv, ok := av.(*types.AttributeValueMemberS)
		if !ok {
			return "", fmt.Errorf("key attribute %q must be string", name)
		}
		b.WriteByte(':')
		b.WriteString(name)
		b.WriteByte('-')
		b.WriteString(sv.Value)
	}
	return b.String(), nil
}
