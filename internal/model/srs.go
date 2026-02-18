package model

import "time"

// SRSState represents the spaced repetition state for a user-item pair
type SRSState struct {
	UserID         string    `json:"user_id" dynamodbav:"user_id"`
	ItemID         string    `json:"item_id" dynamodbav:"item_id"`
	Interval       int       `json:"interval" dynamodbav:"interval"`               // Days until next review
	EaseFactor     float64   `json:"ease_factor" dynamodbav:"ease_factor"`         // Difficulty multiplier (default 2.5)
	DueDate        time.Time `json:"due_date" dynamodbav:"due_date"`               // Next review date
	Repetitions    int       `json:"repetitions" dynamodbav:"repetitions"`         // Number of successful reviews
	LastReviewedAt time.Time `json:"last_reviewed_at" dynamodbav:"last_reviewed_at"`
}

// NewSRSState creates a new SRS state with default values
func NewSRSState(userID, itemID string) *SRSState {
	return &SRSState{
		UserID:         userID,
		ItemID:         itemID,
		Interval:       0,
		EaseFactor:     2.5,
		DueDate:        time.Now(),
		Repetitions:    0,
		LastReviewedAt: time.Time{},
	}
}

// IsDue checks if the item is due for review
func (s *SRSState) IsDue() bool {
	return time.Now().After(s.DueDate) || time.Now().Equal(s.DueDate)
}
