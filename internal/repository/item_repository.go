package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/losts/syun-eng/backend/internal/model"
)

// ItemRepository handles Item persistence
type ItemRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewItemRepository creates a new ItemRepository
func NewItemRepository(client *dynamodb.Client, tableName string) *ItemRepository {
	return &ItemRepository{
		client:    client,
		tableName: tableName,
	}
}

// DynamoDB item structure
type itemRecord struct {
	PK           string   `dynamodbav:"PK"`
	SK           string   `dynamodbav:"SK"`
	ItemID       string   `dynamodbav:"item_id"`
	Japanese     string   `dynamodbav:"japanese"`
	Answers      []string `dynamodbav:"answers"`
	Acceptable   []string `dynamodbav:"acceptable"`
	Situations   []string `dynamodbav:"situations"`
	Patterns     []string `dynamodbav:"patterns"`
	WordCount    int      `dynamodbav:"word_count"`
	LengthBucket string   `dynamodbav:"length_bucket"`
	Difficulty   int      `dynamodbav:"difficulty"`
	CreatedAt    string   `dynamodbav:"created_at"`
}

// GetByID retrieves an item by ID
func (r *ItemRepository) GetByID(ctx context.Context, itemID string) (*model.Item, error) {
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", itemID)},
		"SK": &types.AttributeValueMemberS{Value: "METADATA"},
	}

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		return nil, nil // Not found
	}

	var record itemRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	return r.recordToModel(&record), nil
}

// Query retrieves items matching the given criteria
type ItemQuery struct {
	Situations    []string
	LengthBuckets []model.LengthBucket
	Limit         int
}

// List retrieves items with optional filtering
func (r *ItemRepository) List(ctx context.Context, query *ItemQuery) ([]*model.Item, error) {
	// For DynamoDB, we'll use Scan with filters
	// In production, consider GSIs for better performance
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sk": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	}

	if query != nil && query.Limit > 0 {
		input.Limit = aws.Int32(int32(query.Limit))
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan items: %w", err)
	}

	items := make([]*model.Item, 0, len(result.Items))
	for _, item := range result.Items {
		var record itemRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal item: %w", err)
		}

		modelItem := r.recordToModel(&record)

		// Apply filters in memory (for MVP; consider GSIs for production)
		if r.matchesQuery(modelItem, query) {
			items = append(items, modelItem)
		}
	}

	return items, nil
}

// Create stores a new item
func (r *ItemRepository) Create(ctx context.Context, item *model.Item) error {
	record := r.modelToRecord(item)

	av, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

// BatchCreate stores multiple items
func (r *ItemRepository) BatchCreate(ctx context.Context, items []*model.Item) error {
	// DynamoDB BatchWriteItem supports up to 25 items per request
	const batchSize = 25

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		writeRequests := make([]types.WriteRequest, 0, end-i)
		for _, item := range items[i:end] {
			record := r.modelToRecord(item)
			av, err := attributevalue.MarshalMap(record)
			if err != nil {
				return fmt.Errorf("failed to marshal item: %w", err)
			}

			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{Item: av},
			})
		}

		_, err := r.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				r.tableName: writeRequests,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to batch write items: %w", err)
		}
	}

	return nil
}

func (r *ItemRepository) recordToModel(record *itemRecord) *model.Item {
	return &model.Item{
		ItemID:       record.ItemID,
		Japanese:     record.Japanese,
		Answers:      record.Answers,
		Acceptable:   record.Acceptable,
		Situations:   record.Situations,
		Patterns:     record.Patterns,
		WordCount:    record.WordCount,
		LengthBucket: model.LengthBucket(record.LengthBucket),
		Difficulty:   record.Difficulty,
	}
}

func (r *ItemRepository) modelToRecord(item *model.Item) *itemRecord {
	return &itemRecord{
		PK:           fmt.Sprintf("ITEM#%s", item.ItemID),
		SK:           "METADATA",
		ItemID:       item.ItemID,
		Japanese:     item.Japanese,
		Answers:      item.Answers,
		Acceptable:   item.Acceptable,
		Situations:   item.Situations,
		Patterns:     item.Patterns,
		WordCount:    item.WordCount,
		LengthBucket: string(item.LengthBucket),
		Difficulty:   item.Difficulty,
		CreatedAt:    item.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (r *ItemRepository) matchesQuery(item *model.Item, query *ItemQuery) bool {
	if query == nil {
		return true
	}

	// Check situations
	if len(query.Situations) > 0 {
		found := false
		for _, qs := range query.Situations {
			for _, is := range item.Situations {
				if qs == is {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check length buckets
	if len(query.LengthBuckets) > 0 {
		found := false
		for _, lb := range query.LengthBuckets {
			if item.LengthBucket == lb {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
