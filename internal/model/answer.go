package model

import "time"

// Answer represents a user's answer log entry
type Answer struct {
	UserID         string       `json:"user_id" dynamodbav:"user_id"`
	ItemID         string       `json:"item_id" dynamodbav:"item_id"`
	UserInput      string       `json:"user_input" dynamodbav:"user_input"`
	IsCorrect      bool         `json:"is_correct" dynamodbav:"is_correct"`
	ResponseTimeMs int64        `json:"response_time_ms" dynamodbav:"response_time_ms"`
	Situation      string       `json:"situation" dynamodbav:"situation"`
	LengthBucket   LengthBucket `json:"length_bucket" dynamodbav:"length_bucket"`
	CreatedAt      time.Time    `json:"created_at" dynamodbav:"created_at"`
}
