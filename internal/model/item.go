package model

import "time"

// LengthBucket represents the length category of a question
type LengthBucket string

const (
	LengthS  LengthBucket = "S"  // ~6 words
	LengthM  LengthBucket = "M"  // 7-12 words
	LengthL  LengthBucket = "L"  // 13-20 words
	LengthXL LengthBucket = "XL" // 21+ words
)

// Item represents a practice question
type Item struct {
	ItemID       string       `json:"item_id" dynamodbav:"item_id"`
	Japanese     string       `json:"japanese" dynamodbav:"japanese"`
	Answers      []string     `json:"answers" dynamodbav:"answers"`           // Model answers
	Acceptable   []string     `json:"acceptable" dynamodbav:"acceptable"`     // Acceptable alternatives
	Situations   []string     `json:"situations" dynamodbav:"situations"`     // Situation tags
	Patterns     []string     `json:"patterns" dynamodbav:"patterns"`         // Pattern tags (reporting, request, etc.)
	WordCount    int          `json:"word_count" dynamodbav:"word_count"`
	LengthBucket LengthBucket `json:"length_bucket" dynamodbav:"length_bucket"`
	Difficulty   int          `json:"difficulty" dynamodbav:"difficulty"` // 1-5
	CreatedAt    time.Time    `json:"created_at" dynamodbav:"created_at"`
}

// GetLengthBucket calculates the length bucket from word count
func GetLengthBucket(wordCount int) LengthBucket {
	switch {
	case wordCount <= 6:
		return LengthS
	case wordCount <= 12:
		return LengthM
	case wordCount <= 20:
		return LengthL
	default:
		return LengthXL
	}
}
