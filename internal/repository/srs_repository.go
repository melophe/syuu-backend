package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/losts/syun-eng/backend/internal/model"
)

// SRSRepository handles SRS state persistence
type SRSRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewSRSRepository creates a new SRSRepository
func NewSRSRepository(client *dynamodb.Client, tableName string) *SRSRepository {
	return &SRSRepository{
		client:    client,
		tableName: tableName,
	}
}

type srsRecord struct {
	PK             string  `dynamodbav:"PK"`
	SK             string  `dynamodbav:"SK"`
	UserID         string  `dynamodbav:"user_id"`
	ItemID         string  `dynamodbav:"item_id"`
	Interval       int     `dynamodbav:"interval"`
	EaseFactor     float64 `dynamodbav:"ease_factor"`
	DueDate        string  `dynamodbav:"due_date"`
	Repetitions    int     `dynamodbav:"repetitions"`
	LastReviewedAt string  `dynamodbav:"last_reviewed_at"`
}

// Get retrieves SRS state for a user-item pair
func (r *SRSRepository) Get(ctx context.Context, userID, itemID string) (*model.SRSState, error) {
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
		"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("SRS#%s", itemID)},
	}

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get SRS state: %w", err)
	}

	if result.Item == nil {
		return nil, nil
	}

	var record srsRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SRS state: %w", err)
	}

	return r.recordToModel(&record)
}

// GetDueItems retrieves items due for review for a user
func (r *SRSRepository) GetDueItems(ctx context.Context, userID string, limit int) ([]*model.SRSState, error) {
	now := time.Now().Format("2006-01-02")

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		FilterExpression:       aws.String("due_date <= :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":prefix": &types.AttributeValueMemberS{Value: "SRS#"},
			":now":    &types.AttributeValueMemberS{Value: now},
		},
	}

	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query due items: %w", err)
	}

	states := make([]*model.SRSState, 0, len(result.Items))
	for _, item := range result.Items {
		var record srsRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SRS state: %w", err)
		}

		state, err := r.recordToModel(&record)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

// GetAllForUser retrieves all SRS states for a user
func (r *SRSRepository) GetAllForUser(ctx context.Context, userID string) ([]*model.SRSState, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":prefix": &types.AttributeValueMemberS{Value: "SRS#"},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query SRS states: %w", err)
	}

	states := make([]*model.SRSState, 0, len(result.Items))
	for _, item := range result.Items {
		var record srsRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SRS state: %w", err)
		}

		state, err := r.recordToModel(&record)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

// Upsert creates or updates SRS state
func (r *SRSRepository) Upsert(ctx context.Context, state *model.SRSState) error {
	record := r.modelToRecord(state)

	av, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal SRS state: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put SRS state: %w", err)
	}

	return nil
}

func (r *SRSRepository) recordToModel(record *srsRecord) (*model.SRSState, error) {
	dueDate, _ := time.Parse("2006-01-02", record.DueDate)
	lastReviewedAt, _ := time.Parse(time.RFC3339, record.LastReviewedAt)

	return &model.SRSState{
		UserID:         record.UserID,
		ItemID:         record.ItemID,
		Interval:       record.Interval,
		EaseFactor:     record.EaseFactor,
		DueDate:        dueDate,
		Repetitions:    record.Repetitions,
		LastReviewedAt: lastReviewedAt,
	}, nil
}

func (r *SRSRepository) modelToRecord(state *model.SRSState) *srsRecord {
	return &srsRecord{
		PK:             fmt.Sprintf("USER#%s", state.UserID),
		SK:             fmt.Sprintf("SRS#%s", state.ItemID),
		UserID:         state.UserID,
		ItemID:         state.ItemID,
		Interval:       state.Interval,
		EaseFactor:     state.EaseFactor,
		DueDate:        state.DueDate.Format("2006-01-02"),
		Repetitions:    state.Repetitions,
		LastReviewedAt: state.LastReviewedAt.Format(time.RFC3339),
	}
}
