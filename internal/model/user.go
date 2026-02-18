package model

import "time"

// User represents an authenticated user
type User struct {
	UserID      string    `json:"user_id" dynamodbav:"user_id"`
	Email       string    `json:"email" dynamodbav:"email"`
	Name        string    `json:"name" dynamodbav:"name"`
	GoogleID    string    `json:"google_id" dynamodbav:"google_id"`
	CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
	LastLoginAt time.Time `json:"last_login_at" dynamodbav:"last_login_at"`
}
