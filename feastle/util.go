package feastle

import (
	"fmt"
	"math/rand"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type FeastFeature struct {
	FeatureName    string
	EntityId       string
	EventTimestamp string
	Values         map[string][]byte
}

func GenerateRandomFeature(featureNames []string) *FeastFeature {
	featureName := featureNames[rand.Intn(len(featureNames))]
	randId := fmt.Sprintf("key-%d", rand.Int()%1000)
	randTs := fmt.Sprintf("ts-%d", rand.Int()%1000)
	randValue1 := fmt.Sprintf("%d", rand.Int()%100000)
	randValue2 := fmt.Sprintf("%d", rand.Int()%100000)
	randValue3 := fmt.Sprintf("%d", rand.Int()%100000)

	return &FeastFeature{
		FeatureName:    featureName,
		EntityId:       randId,
		EventTimestamp: randTs,
		Values: map[string][]byte{
			"key1": []byte(randValue1),
			"key2": []byte(randValue2),
			"key3": []byte(randValue3),
		},
	}
}

func (f *FeastFeature) ddbItem() map[string]types.AttributeValue {
	values := map[string]types.AttributeValue{}
	for k, v := range f.Values {
		values[k] = &types.AttributeValueMemberB{Value: v}
	}
	item := map[string]types.AttributeValue{
		"entity_id": &types.AttributeValueMemberS{Value: f.EntityId},
		"event_ts":  &types.AttributeValueMemberS{Value: f.EventTimestamp},
		"values":    &types.AttributeValueMemberM{Value: values},
	}
	return item
}

func NewBatchWriteItemInput(features []*FeastFeature) *dynamodb.BatchWriteItemInput {
	requestItems := map[string][]types.WriteRequest{}
	for _, f := range features {
		requestItems[f.FeatureName] = append(requestItems[f.FeatureName], types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: f.ddbItem(),
			},
		})
	}
	return &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	}
}

func NewBatchWriteItemInputDelete(features []*FeastFeature) *dynamodb.BatchWriteItemInput {
	requestItems := map[string][]types.WriteRequest{}
	for _, f := range features {
		requestItems[f.FeatureName] = append(requestItems[f.FeatureName], types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					"entity_id": &types.AttributeValueMemberS{Value: f.EntityId},
				},
			},
		})
	}
	return &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	}
}

func NewBatchGetItemInput(features []*FeastFeature) *dynamodb.BatchGetItemInput {
	requestItems := map[string]types.KeysAndAttributes{}
	for _, f := range features {
		requestItems[f.FeatureName] = types.KeysAndAttributes{
			Keys: []map[string]types.AttributeValue{
				{
					"entity_id": &types.AttributeValueMemberS{Value: f.EntityId},
				},
			},
		}
	}

	return &dynamodb.BatchGetItemInput{
		RequestItems: requestItems,
	}
}

func GenerateRandomBatchWrite(tables []string, batchSize int) *dynamodb.BatchWriteItemInput {
	randomFeatures := []*FeastFeature{}
	for i := 0; i < batchSize; i++ {
		randomFeatures = append(randomFeatures, GenerateRandomFeature(tables))
	}
	return NewBatchWriteItemInput(randomFeatures)
}

func GenerateRandomBatchWriteDelete(tables []string, batchSize int) *dynamodb.BatchWriteItemInput {
	randomFeatures := []*FeastFeature{}
	for i := 0; i < batchSize; i++ {
		randomFeatures = append(randomFeatures, GenerateRandomFeature(tables))
	}
	return NewBatchWriteItemInputDelete(randomFeatures)
}
