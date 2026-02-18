package srs

import (
	"testing"
	"time"

	"github.com/losts/syun-eng/backend/internal/model"
)

func TestUpdate_FirstCorrectReview(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")

	a.Update(state, QualityCorrectNormal)

	if state.Interval != 1 {
		t.Errorf("Expected interval 1, got %d", state.Interval)
	}
	if state.Repetitions != 1 {
		t.Errorf("Expected repetitions 1, got %d", state.Repetitions)
	}
}

func TestUpdate_SecondCorrectReview(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")

	a.Update(state, QualityCorrectNormal)
	a.Update(state, QualityCorrectNormal)

	if state.Interval != 6 {
		t.Errorf("Expected interval 6, got %d", state.Interval)
	}
	if state.Repetitions != 2 {
		t.Errorf("Expected repetitions 2, got %d", state.Repetitions)
	}
}

func TestUpdate_ThirdCorrectReview(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")

	a.Update(state, QualityCorrectNormal)
	a.Update(state, QualityCorrectNormal)
	a.Update(state, QualityCorrectNormal)

	// Third review: interval = 6 * ease_factor (approximately 2.5)
	// Should be around 15
	if state.Interval < 14 || state.Interval > 16 {
		t.Errorf("Expected interval around 15, got %d", state.Interval)
	}
	if state.Repetitions != 3 {
		t.Errorf("Expected repetitions 3, got %d", state.Repetitions)
	}
}

func TestUpdate_IncorrectResetRepetitions(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")

	// Do some correct reviews first
	a.Update(state, QualityCorrectNormal)
	a.Update(state, QualityCorrectNormal)
	a.Update(state, QualityCorrectNormal)

	// Now fail
	a.Update(state, QualityIncorrect)

	if state.Repetitions != 0 {
		t.Errorf("Expected repetitions to reset to 0, got %d", state.Repetitions)
	}
	if state.Interval != 1 {
		t.Errorf("Expected interval to reset to 1, got %d", state.Interval)
	}
}

func TestUpdate_EaseFactorMinimum(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")

	// Repeatedly get answers wrong to lower ease factor
	for i := 0; i < 10; i++ {
		a.Update(state, QualityCorrectHard) // Minimum correct quality
	}

	if state.EaseFactor < 1.3 {
		t.Errorf("Ease factor should not go below 1.3, got %f", state.EaseFactor)
	}
}

func TestUpdate_EaseFactorIncreasesWithPerfect(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")
	initialEase := state.EaseFactor

	a.Update(state, QualityCorrectPerfect)

	if state.EaseFactor <= initialEase {
		t.Errorf("Ease factor should increase with perfect recall")
	}
}

func TestUpdate_DueDateSet(t *testing.T) {
	a := New()
	state := model.NewSRSState("user1", "item1")
	before := time.Now()

	a.Update(state, QualityCorrectNormal)

	if state.DueDate.Before(before) {
		t.Errorf("Due date should be in the future")
	}

	expectedDue := before.AddDate(0, 0, 1)
	// Allow some tolerance for test execution time
	if state.DueDate.Sub(expectedDue).Hours() > 1 {
		t.Errorf("Due date should be approximately 1 day from now")
	}
}

func TestGetQualityFromCorrectness(t *testing.T) {
	tests := []struct {
		name           string
		isCorrect      bool
		responseTimeMs int64
		expected       Quality
	}{
		{"incorrect", false, 1000, QualityIncorrect},
		{"fast correct", true, 2000, QualityCorrectPerfect},
		{"normal correct", true, 5000, QualityCorrectNormal},
		{"slow correct", true, 15000, QualityCorrectHard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetQualityFromCorrectness(tt.isCorrect, tt.responseTimeMs)
			if got != tt.expected {
				t.Errorf("GetQualityFromCorrectness(%v, %d) = %d, want %d",
					tt.isCorrect, tt.responseTimeMs, got, tt.expected)
			}
		})
	}
}

func TestCalculatePriority_NewItem(t *testing.T) {
	a := New()
	priority := a.CalculatePriority(nil)

	if priority != 100.0 {
		t.Errorf("New item priority should be 100, got %f", priority)
	}
}

func TestCalculatePriority_OverdueItem(t *testing.T) {
	a := New()
	state := &model.SRSState{
		UserID:  "user1",
		ItemID:  "item1",
		DueDate: time.Now().AddDate(0, 0, -2), // 2 days overdue
	}

	priority := a.CalculatePriority(state)

	if priority <= 50.0 {
		t.Errorf("Overdue item should have priority > 50, got %f", priority)
	}
}

func TestCalculatePriority_FutureItem(t *testing.T) {
	a := New()
	state := &model.SRSState{
		UserID:  "user1",
		ItemID:  "item1",
		DueDate: time.Now().AddDate(0, 0, 5), // 5 days in future
	}

	priority := a.CalculatePriority(state)

	if priority >= 50.0 {
		t.Errorf("Future item should have priority < 50, got %f", priority)
	}
}
