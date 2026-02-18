package service

import (
	"context"
	"time"

	"github.com/losts/syun-eng/backend/internal/model"
	"github.com/losts/syun-eng/backend/internal/repository"
)

// StatsService handles statistics
type StatsService struct {
	answerRepo *repository.AnswerRepository
	srsRepo    *repository.SRSRepository
}

// NewStatsService creates a new StatsService
func NewStatsService(answerRepo *repository.AnswerRepository, srsRepo *repository.SRSRepository) *StatsService {
	return &StatsService{
		answerRepo: answerRepo,
		srsRepo:    srsRepo,
	}
}

// Summary represents overall statistics
type Summary struct {
	TotalAnswers     int                              `json:"total_answers"`
	CorrectAnswers   int                              `json:"correct_answers"`
	AccuracyRate     float64                          `json:"accuracy_rate"`
	AvgResponseMs    int64                            `json:"avg_response_ms"`
	DueReviewCount   int                              `json:"due_review_count"`
	ByLengthBucket   map[model.LengthBucket]BucketStats `json:"by_length_bucket"`
	BySituation      map[string]BucketStats           `json:"by_situation"`
	TodayAnswers     int                              `json:"today_answers"`
	TodayCorrect     int                              `json:"today_correct"`
	WeekAnswers      int                              `json:"week_answers"`
	WeekCorrect      int                              `json:"week_correct"`
}

// BucketStats represents stats for a category
type BucketStats struct {
	Total    int     `json:"total"`
	Correct  int     `json:"correct"`
	Accuracy float64 `json:"accuracy"`
}

// GetSummary retrieves overall statistics for a user
func (s *StatsService) GetSummary(ctx context.Context, userID string) (*Summary, error) {
	// Get all-time stats
	allTimeStats, err := s.answerRepo.GetStats(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	// Get today's stats
	today := time.Now().Truncate(24 * time.Hour)
	todayStats, err := s.answerRepo.GetStats(ctx, userID, &today)
	if err != nil {
		return nil, err
	}

	// Get this week's stats
	weekAgo := time.Now().AddDate(0, 0, -7)
	weekStats, err := s.answerRepo.GetStats(ctx, userID, &weekAgo)
	if err != nil {
		return nil, err
	}

	// Get due review count
	dueItems, err := s.srsRepo.GetDueItems(ctx, userID, 0)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		TotalAnswers:   allTimeStats.TotalAnswers,
		CorrectAnswers: allTimeStats.CorrectAnswers,
		AvgResponseMs:  allTimeStats.AvgResponseMs,
		DueReviewCount: len(dueItems),
		TodayAnswers:   todayStats.TotalAnswers,
		TodayCorrect:   todayStats.CorrectAnswers,
		WeekAnswers:    weekStats.TotalAnswers,
		WeekCorrect:    weekStats.CorrectAnswers,
	}

	if summary.TotalAnswers > 0 {
		summary.AccuracyRate = float64(summary.CorrectAnswers) / float64(summary.TotalAnswers) * 100
	}

	// Convert length bucket stats
	summary.ByLengthBucket = make(map[model.LengthBucket]BucketStats)
	for lb, stats := range allTimeStats.ByLengthBucket {
		accuracy := 0.0
		if stats.Total > 0 {
			accuracy = float64(stats.Correct) / float64(stats.Total) * 100
		}
		summary.ByLengthBucket[lb] = BucketStats{
			Total:    stats.Total,
			Correct:  stats.Correct,
			Accuracy: accuracy,
		}
	}

	// Convert situation stats
	summary.BySituation = make(map[string]BucketStats)
	for sit, stats := range allTimeStats.BySituation {
		accuracy := 0.0
		if stats.Total > 0 {
			accuracy = float64(stats.Correct) / float64(stats.Total) * 100
		}
		summary.BySituation[sit] = BucketStats{
			Total:    stats.Total,
			Correct:  stats.Correct,
			Accuracy: accuracy,
		}
	}

	return summary, nil
}

// Weakness represents a weak area
type Weakness struct {
	Category   string  `json:"category"`
	Type       string  `json:"type"` // "situation" or "length"
	Total      int     `json:"total"`
	Correct    int     `json:"correct"`
	Accuracy   float64 `json:"accuracy"`
}

// GetWeaknesses analyzes and returns weak areas
func (s *StatsService) GetWeaknesses(ctx context.Context, userID string) ([]Weakness, error) {
	stats, err := s.answerRepo.GetStats(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	weaknesses := make([]Weakness, 0)

	// Analyze by situation
	for sit, data := range stats.BySituation {
		if data.Total >= 5 { // Minimum sample size
			accuracy := float64(data.Correct) / float64(data.Total) * 100
			if accuracy < 70 { // Below 70% is considered weak
				weaknesses = append(weaknesses, Weakness{
					Category: sit,
					Type:     "situation",
					Total:    data.Total,
					Correct:  data.Correct,
					Accuracy: accuracy,
				})
			}
		}
	}

	// Analyze by length bucket
	for lb, data := range stats.ByLengthBucket {
		if data.Total >= 5 {
			accuracy := float64(data.Correct) / float64(data.Total) * 100
			if accuracy < 70 {
				weaknesses = append(weaknesses, Weakness{
					Category: string(lb),
					Type:     "length",
					Total:    data.Total,
					Correct:  data.Correct,
					Accuracy: accuracy,
				})
			}
		}
	}

	return weaknesses, nil
}
