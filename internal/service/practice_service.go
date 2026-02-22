package service

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/losts/syun-eng/backend/internal/model"
	"github.com/losts/syun-eng/backend/internal/repository"
	"github.com/losts/syun-eng/backend/pkg/scorer"
	"github.com/losts/syun-eng/backend/pkg/srs"
)

// PracticeService handles practice session logic
type PracticeService struct {
	itemRepo   *repository.ItemRepository
	srsRepo    *repository.SRSRepository
	answerRepo *repository.AnswerRepository
	generator  *GeneratorService
	scorer     *scorer.Scorer
	srsAlgo    *srs.Algorithm

	// In-memory session storage (use Redis in production)
	sessions       map[string]*model.PracticeSession
	generatedItems map[string]*model.Item // Temporary storage for AI-generated items
}

// NewPracticeService creates a new PracticeService
func NewPracticeService(
	itemRepo *repository.ItemRepository,
	srsRepo *repository.SRSRepository,
	answerRepo *repository.AnswerRepository,
	generator *GeneratorService,
) *PracticeService {
	return &PracticeService{
		itemRepo:       itemRepo,
		srsRepo:        srsRepo,
		answerRepo:     answerRepo,
		generator:      generator,
		scorer:         scorer.New(),
		srsAlgo:        srs.New(),
		sessions:       make(map[string]*model.PracticeSession),
		generatedItems: make(map[string]*model.Item),
	}
}

// StartSession starts a new practice session
func (s *PracticeService) StartSession(ctx context.Context, userID string, settings model.PracticeSettings) (*model.PracticeSession, error) {
	totalQuestions := settings.QuestionCount
	if totalQuestions <= 0 {
		totalQuestions = 10
	}

	var reviewItemIDs []string
	var reviewCount int

	// Get due review items from DB
	query := &repository.ItemQuery{
		UserID:        userID,
		Situations:    settings.Situations,
		LengthBuckets: settings.LengthBuckets,
	}

	items, err := s.itemRepo.ListByUser(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	if len(items) > 0 {
		// Get SRS states to find due items
		srsStates, err := s.srsRepo.GetAllForUser(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get SRS states: %w", err)
		}

		srsMap := make(map[string]*model.SRSState)
		for _, state := range srsStates {
			srsMap[state.ItemID] = state
		}

		// Sort items by priority (due items first)
		reviewItemIDs = s.prioritizeItems(items, srsMap, true)

		if settings.ReviewPriority {
			// Review priority mode: 100% review
			reviewCount = totalQuestions
		} else {
			// Hybrid mode: 30% review, 70% new
			reviewCount = totalQuestions * 30 / 100
		}

		// Limit review items
		if len(reviewItemIDs) > reviewCount {
			reviewItemIDs = reviewItemIDs[:reviewCount]
		}
		reviewCount = len(reviewItemIDs)
	}

	// Calculate how many new items to generate
	newCount := totalQuestions - reviewCount

	session := &model.PracticeSession{
		SessionID:       uuid.New().String(),
		UserID:          userID,
		Settings:        settings,
		CurrentIndex:    0,
		TotalQuestions:  totalQuestions,
		CorrectCount:    0,
		ItemIDs:         reviewItemIDs,        // Review items (pre-loaded)
		NewItemCount:    newCount,             // New items to generate
		AnsweredItemIDs: []string{},
	}

	s.sessions[session.SessionID] = session

	return session, nil
}

// GetNextQuestion gets the next question in the session
func (s *PracticeService) GetNextQuestion(ctx context.Context, sessionID string) (*model.PracticeQuestion, error) {
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if session.CurrentIndex >= session.TotalQuestions {
		return nil, nil // Session complete
	}

	var item *model.Item
	var err error

	// Determine if this question should be review or new
	// Interleave: every 3rd question is review (if available)
	reviewItemsRemaining := len(session.ItemIDs)
	newItemsRemaining := session.NewItemCount
	shouldUseReview := false

	if reviewItemsRemaining > 0 && newItemsRemaining > 0 {
		// Interleave pattern: R, N, N, R, N, N, ...
		shouldUseReview = (session.CurrentIndex % 3) == 0
	} else if reviewItemsRemaining > 0 {
		shouldUseReview = true
	}

	if shouldUseReview && len(session.ItemIDs) > 0 {
		// Get review item from DB
		itemID := session.ItemIDs[0]
		session.ItemIDs = session.ItemIDs[1:] // Remove from list

		item = s.getGeneratedItem(sessionID, itemID)
		if item == nil {
			item, err = s.itemRepo.GetByID(ctx, itemID)
			if err != nil {
				return nil, fmt.Errorf("failed to get item: %w", err)
			}
		}
	}

	// Generate new item with AI if needed
	if item == nil && s.generator != nil && s.generator.IsEnabled() && session.NewItemCount > 0 {
		lengthBucket := model.LengthM
		if len(session.Settings.LengthBuckets) > 0 {
			lengthBucket = session.Settings.LengthBuckets[rand.Intn(len(session.Settings.LengthBuckets))]
		}

		item, err = s.generator.GenerateItem(ctx, session.Settings.Situations, lengthBucket, session.Settings.CustomTopic)
		if err != nil {
			return nil, fmt.Errorf("failed to generate item: %w", err)
		}

		// Set user ID and custom topic for later DB save
		item.UserID = session.UserID
		item.CustomTopic = session.Settings.CustomTopic

		// Store generated item for answer checking
		s.storeGeneratedItem(sessionID, item)

		// Decrement new item count
		session.NewItemCount--
	}

	if item == nil {
		return nil, fmt.Errorf("no item available")
	}

	return &model.PracticeQuestion{
		ItemID:         item.ItemID,
		Japanese:       item.Japanese,
		LengthBucket:   item.LengthBucket,
		Difficulty:     item.Difficulty,
		QuestionNumber: session.CurrentIndex + 1,
		TotalQuestions: session.TotalQuestions,
	}, nil
}

// SubmitAnswer submits an answer and returns the result
func (s *PracticeService) SubmitAnswer(ctx context.Context, sessionID, itemID, userInput string, responseTimeMs int64) (*model.AnswerResult, error) {
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// Get the item (check generated items first, then DB)
	item := s.getGeneratedItem(sessionID, itemID)
	isGenerated := item != nil
	if item == nil {
		var err error
		item, err = s.itemRepo.GetByID(ctx, itemID)
		if err != nil {
			return nil, fmt.Errorf("failed to get item: %w", err)
		}
	}
	if item == nil {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}

	// Score the answer
	scoreResult := s.scorer.Score(userInput, item.Answers, item.Acceptable)

	// Update session
	session.CurrentIndex++
	session.AnsweredItemIDs = append(session.AnsweredItemIDs, itemID)
	if scoreResult.IsCorrect {
		session.CorrectCount++
	}

	// Save AI-generated item to DB for future review
	if isGenerated {
		item.CreatedAt = time.Now()
		if err := s.itemRepo.Create(ctx, item); err != nil {
			// Log error but don't fail the answer submission
			fmt.Printf("failed to save generated item: %v\n", err)
		}
	}

	// Update SRS state
	srsState, err := s.srsRepo.Get(ctx, session.UserID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get SRS state: %w", err)
	}
	if srsState == nil {
		srsState = model.NewSRSState(session.UserID, itemID)
	}

	quality := srs.GetQualityFromCorrectness(scoreResult.IsCorrect, responseTimeMs)
	s.srsAlgo.Update(srsState, quality)

	if err := s.srsRepo.Upsert(ctx, srsState); err != nil {
		return nil, fmt.Errorf("failed to update SRS state: %w", err)
	}

	// Save answer log
	situation := ""
	if len(item.Situations) > 0 {
		situation = item.Situations[0]
	}

	answer := &model.Answer{
		UserID:         session.UserID,
		ItemID:         itemID,
		UserInput:      userInput,
		IsCorrect:      scoreResult.IsCorrect,
		ResponseTimeMs: responseTimeMs,
		Situation:      situation,
		LengthBucket:   item.LengthBucket,
		CreatedAt:      time.Now(),
	}

	if err := s.answerRepo.Create(ctx, answer); err != nil {
		return nil, fmt.Errorf("failed to save answer: %w", err)
	}

	return &model.AnswerResult{
		IsCorrect:    scoreResult.IsCorrect,
		UserInput:    userInput,
		ModelAnswers: item.Answers,
		Acceptable:   item.Acceptable,
		MatchedWith:  scoreResult.MatchedWith,
	}, nil
}

// GetSession retrieves a session by ID
func (s *PracticeService) GetSession(sessionID string) (*model.PracticeSession, error) {
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// GetHint returns a hint for a given item
func (s *PracticeService) GetHint(ctx context.Context, sessionID, itemID string, level int) (string, error) {
	// Get the item (check generated items first, then DB)
	item := s.getGeneratedItem(sessionID, itemID)
	if item == nil {
		var err error
		item, err = s.itemRepo.GetByID(ctx, itemID)
		if err != nil {
			return "", fmt.Errorf("failed to get item: %w", err)
		}
	}
	if item == nil {
		return "", fmt.Errorf("item not found: %s", itemID)
	}

	return s.generator.GenerateHint(item, level), nil
}

// EndSession ends a practice session
func (s *PracticeService) EndSession(sessionID string) {
	// Clean up generated items for this session
	prefix := sessionID + ":"
	for key := range s.generatedItems {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.generatedItems, key)
		}
	}
	delete(s.sessions, sessionID)
}

// storeGeneratedItem temporarily stores an AI-generated item
func (s *PracticeService) storeGeneratedItem(sessionID string, item *model.Item) {
	key := sessionID + ":" + item.ItemID
	s.generatedItems[key] = item
}

// getGeneratedItem retrieves a temporarily stored AI-generated item
func (s *PracticeService) getGeneratedItem(sessionID, itemID string) *model.Item {
	key := sessionID + ":" + itemID
	return s.generatedItems[key]
}

type itemWithPriority struct {
	itemID   string
	priority float64
}

func (s *PracticeService) prioritizeItems(items []*model.Item, srsMap map[string]*model.SRSState, reviewPriority bool) []string {
	itemsWithPriority := make([]itemWithPriority, len(items))

	for i, item := range items {
		state := srsMap[item.ItemID]
		priority := s.srsAlgo.CalculatePriority(state)

		// Add some randomness to avoid predictable order
		priority += rand.Float64() * 5

		itemsWithPriority[i] = itemWithPriority{
			itemID:   item.ItemID,
			priority: priority,
		}
	}

	// Sort by priority (highest first)
	if reviewPriority {
		sort.Slice(itemsWithPriority, func(i, j int) bool {
			return itemsWithPriority[i].priority > itemsWithPriority[j].priority
		})
	} else {
		// Shuffle for new items priority
		rand.Shuffle(len(itemsWithPriority), func(i, j int) {
			itemsWithPriority[i], itemsWithPriority[j] = itemsWithPriority[j], itemsWithPriority[i]
		})
	}

	result := make([]string, len(itemsWithPriority))
	for i, item := range itemsWithPriority {
		result[i] = item.itemID
	}

	return result
}
