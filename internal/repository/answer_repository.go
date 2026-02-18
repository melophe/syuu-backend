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

// AnswerRepository handles Answer persistence
type AnswerRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewAnswerRepository creates a new AnswerRepository
func NewAnswerRepository(client *dynamodb.Client, tableName string) *AnswerRepository {
	return &AnswerRepository{
		client:    client,
		tableName: tableName,
	}
}

type answerRecord struct {
	PK             string `dynamodbav:"PK"`
	SK             string `dynamodbav:"SK"`
	UserID         string `dynamodbav:"user_id"`
	ItemID         string `dynamodbav:"item_id"`
	UserInput      string `dynamodbav:"user_input"`
	IsCorrect      bool   `dynamodbav:"is_correct"`
	ResponseTimeMs int64  `dynamodbav:"response_time_ms"`
	Situation      string `dynamodbav:"situation"`
	LengthBucket   string `dynamodbav:"length_bucket"`
	CreatedAt      string `dynamodbav:"created_at"`
}

// Create stores a new answer
func (r *AnswerRepository) Create(ctx context.Context, answer *model.Answer) error {
	record := r.modelToRecord(answer)

	av, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal answer: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put answer: %w", err)
	}

	return nil
}

// GetForUser retrieves answers for a user with optional time range
func (r *AnswerRepository) GetForUser(ctx context.Context, userID string, since *time.Time, limit int) ([]*model.Answer, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":prefix": &types.AttributeValueMemberS{Value: "ANSWER#"},
		},
		ScanIndexForward: aws.Bool(false), // Most recent first
	}

	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query answers: %w", err)
	}

	answers := make([]*model.Answer, 0, len(result.Items))
	for _, item := range result.Items {
		var record answerRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal answer: %w", err)
		}

		answer, err := r.recordToModel(&record)
		if err != nil {
			return nil, err
		}

		// Filter by time if specified
		if since != nil && answer.CreatedAt.Before(*since) {
			continue
		}

		answers = append(answers, answer)
	}

	return answers, nil
}

// GetStats retrieves answer statistics for a user
type AnswerStats struct {
	TotalAnswers   int
	CorrectAnswers int
	AvgResponseMs  int64
	ByLengthBucket map[model.LengthBucket]struct {
		Total   int
		Correct int
	}
	BySituation map[string]struct {
		Total   int
		Correct int
	}
}

func (r *AnswerRepository) GetStats(ctx context.Context, userID string, since *time.Time) (*AnswerStats, error) {
	answers, err := r.GetForUser(ctx, userID, since, 0)
	if err != nil {
		return nil, err
	}

	stats := &AnswerStats{
		ByLengthBucket: make(map[model.LengthBucket]struct {
			Total   int
			Correct int
		}),
		BySituation: make(map[string]struct {
			Total   int
			Correct int
		}),
	}

	var totalResponseTime int64
	for _, answer := range answers {
		stats.TotalAnswers++
		totalResponseTime += answer.ResponseTimeMs

		if answer.IsCorrect {
			stats.CorrectAnswers++
		}

		// By length bucket
		lb := stats.ByLengthBucket[answer.LengthBucket]
		lb.Total++
		if answer.IsCorrect {
			lb.Correct++
		}
		stats.ByLengthBucket[answer.LengthBucket] = lb

		// By situation
		sit := stats.BySituation[answer.Situation]
		sit.Total++
		if answer.IsCorrect {
			sit.Correct++
		}
		stats.BySituation[answer.Situation] = sit
	}

	if stats.TotalAnswers > 0 {
		stats.AvgResponseMs = totalResponseTime / int64(stats.TotalAnswers)
	}

	return stats, nil
}

func (r *AnswerRepository) recordToModel(record *answerRecord) (*model.Answer, error) {
	createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

	return &model.Answer{
		UserID:         record.UserID,
		ItemID:         record.ItemID,
		UserInput:      record.UserInput,
		IsCorrect:      record.IsCorrect,
		ResponseTimeMs: record.ResponseTimeMs,
		Situation:      record.Situation,
		LengthBucket:   model.LengthBucket(record.LengthBucket),
		CreatedAt:      createdAt,
	}, nil
}

func (r *AnswerRepository) modelToRecord(answer *model.Answer) *answerRecord {
	timestamp := answer.CreatedAt.Format("2006-01-02T15:04:05.000Z")
	return &answerRecord{
		PK:             fmt.Sprintf("USER#%s", answer.UserID),
		SK:             fmt.Sprintf("ANSWER#%s#%s", timestamp, answer.ItemID),
		UserID:         answer.UserID,
		ItemID:         answer.ItemID,
		UserInput:      answer.UserInput,
		IsCorrect:      answer.IsCorrect,
		ResponseTimeMs: answer.ResponseTimeMs,
		Situation:      answer.Situation,
		LengthBucket:   string(answer.LengthBucket),
		CreatedAt:      answer.CreatedAt.Format(time.RFC3339),
	}
}
