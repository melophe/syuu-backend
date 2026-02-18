package srs

import (
	"time"

	"github.com/losts/syun-eng/backend/internal/model"
)

// Quality represents the quality of recall (SM-2 algorithm)
type Quality int

const (
	QualityBlackout       Quality = 0 // Complete blackout
	QualityIncorrect      Quality = 1 // Incorrect, but upon seeing the answer, remembered
	QualityIncorrectHard  Quality = 2 // Incorrect, but seemed easy to recall
	QualityCorrectHard    Quality = 3 // Correct with serious difficulty
	QualityCorrectNormal  Quality = 4 // Correct with some hesitation
	QualityCorrectPerfect Quality = 5 // Correct with perfect recall
)

// Algorithm implements the SM-2 spaced repetition algorithm
type Algorithm struct{}

// New creates a new SRS Algorithm instance
func New() *Algorithm {
	return &Algorithm{}
}

// Update updates the SRS state based on the answer quality
// quality: 0-5 (0-2: incorrect, 3-5: correct)
func (a *Algorithm) Update(state *model.SRSState, quality Quality) {
	now := time.Now()
	state.LastReviewedAt = now

	if quality >= QualityCorrectHard {
		// Correct response
		switch state.Repetitions {
		case 0:
			state.Interval = 1
		case 1:
			state.Interval = 6
		default:
			state.Interval = int(float64(state.Interval) * state.EaseFactor)
		}
		state.Repetitions++
	} else {
		// Incorrect response - reset
		state.Repetitions = 0
		state.Interval = 1
	}

	// Update ease factor using SM-2 formula
	q := float64(quality)
	state.EaseFactor = state.EaseFactor + (0.1 - (5-q)*(0.08+(5-q)*0.02))

	// Ensure ease factor doesn't go below 1.3
	if state.EaseFactor < 1.3 {
		state.EaseFactor = 1.3
	}

	// Set next due date
	state.DueDate = now.AddDate(0, 0, state.Interval)
}

// GetQualityFromCorrectness converts a simple correct/incorrect to a quality score
// For MVP, we use a simplified model
func GetQualityFromCorrectness(isCorrect bool, responseTimeMs int64) Quality {
	if !isCorrect {
		return QualityIncorrect
	}

	// Use response time to estimate difficulty
	// < 3 seconds: perfect recall
	// 3-10 seconds: normal recall
	// > 10 seconds: hard recall
	switch {
	case responseTimeMs < 3000:
		return QualityCorrectPerfect
	case responseTimeMs < 10000:
		return QualityCorrectNormal
	default:
		return QualityCorrectHard
	}
}

// CalculatePriority calculates the priority score for an item
// Higher score = higher priority for review
func (a *Algorithm) CalculatePriority(state *model.SRSState) float64 {
	if state == nil {
		return 100.0 // New items get high priority
	}

	now := time.Now()
	dueDate := state.DueDate

	if now.After(dueDate) {
		// Overdue - higher priority based on how overdue
		daysPastDue := now.Sub(dueDate).Hours() / 24
		return 50.0 + daysPastDue*10.0 // Cap at reasonable values in practice
	}

	// Not yet due - lower priority
	daysUntilDue := dueDate.Sub(now).Hours() / 24
	return 50.0 - daysUntilDue*5.0 // Can go negative, which is fine
}
