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

// UserRepository handles User persistence
type UserRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(client *dynamodb.Client, tableName string) *UserRepository {
	return &UserRepository{
		client:    client,
		tableName: tableName,
	}
}

type userRecord struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	UserID      string `dynamodbav:"user_id"`
	Email       string `dynamodbav:"email"`
	Name        string `dynamodbav:"name"`
	GoogleID    string `dynamodbav:"google_id"`
	CreatedAt   string `dynamodbav:"created_at"`
	LastLoginAt string `dynamodbav:"last_login_at"`
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
		"SK": &types.AttributeValueMemberS{Value: "PROFILE"},
	}

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if result.Item == nil {
		return nil, nil
	}

	var record userRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return r.recordToModel(&record)
}

// GetByGoogleID retrieves a user by Google ID
func (r *UserRepository) GetByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	// Use GSI in production; for MVP, scan
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("google_id = :gid AND SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gid": &types.AttributeValueMemberS{Value: googleID},
			":sk":  &types.AttributeValueMemberS{Value: "PROFILE"},
		},
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan users: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var record userRecord
	if err := attributevalue.UnmarshalMap(result.Items[0], &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return r.recordToModel(&record)
}

// Create stores a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	record := r.modelToRecord(user)

	av, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put user: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "PROFILE"},
		},
		UpdateExpression: aws.String("SET last_login_at = :t"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":t": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

func (r *UserRepository) recordToModel(record *userRecord) (*model.User, error) {
	createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)
	lastLoginAt, _ := time.Parse(time.RFC3339, record.LastLoginAt)

	return &model.User{
		UserID:      record.UserID,
		Email:       record.Email,
		Name:        record.Name,
		GoogleID:    record.GoogleID,
		CreatedAt:   createdAt,
		LastLoginAt: lastLoginAt,
	}, nil
}

func (r *UserRepository) modelToRecord(user *model.User) *userRecord {
	return &userRecord{
		PK:          fmt.Sprintf("USER#%s", user.UserID),
		SK:          "PROFILE",
		UserID:      user.UserID,
		Email:       user.Email,
		Name:        user.Name,
		GoogleID:    user.GoogleID,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		LastLoginAt: user.LastLoginAt.Format(time.RFC3339),
	}
}
