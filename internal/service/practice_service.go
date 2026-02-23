package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
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
	mu             sync.RWMutex
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
	rand.Seed(time.Now().UnixNano())
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

	s.mu.Lock()
	s.sessions[session.SessionID] = session
	s.mu.Unlock()

	return session, nil
}

// GetNextQuestion gets the next question in the session
func (s *PracticeService) GetNextQuestion(ctx context.Context, sessionID string) (*model.PracticeQuestion, error) {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("session not found")
	}

	if session.CurrentIndex >= session.TotalQuestions {
		s.mu.Unlock()
		return nil, nil // Session complete
	}

	var item *model.Item
	var err error
	var itemID string

	// Determine if this question should be review or new
	reviewItemsRemaining := len(session.ItemIDs)
	newItemsRemaining := session.NewItemCount
	shouldUseReview := false

	if reviewItemsRemaining > 0 && newItemsRemaining > 0 {
		shouldUseReview = (session.CurrentIndex % 3) == 0
	} else if reviewItemsRemaining > 0 {
		shouldUseReview = true
	}

	if shouldUseReview && len(session.ItemIDs) > 0 {
		itemID = session.ItemIDs[0]
		session.ItemIDs = session.ItemIDs[1:]
	}

	userID := session.UserID
	settings := session.Settings
	needsGeneration := !shouldUseReview || itemID == ""
	if needsGeneration && session.NewItemCount > 0 {
		session.NewItemCount--
	}
	s.mu.Unlock()

	// Fetch or generate item outside the lock
	if itemID != "" {
		item = s.getGeneratedItem(sessionID, itemID)
		if item == nil {
			item, err = s.itemRepo.GetByID(ctx, userID, itemID)
			if err != nil {
				return nil, fmt.Errorf("failed to get item: %w", err)
			}
		}
	}

	if item == nil && s.generator != nil && s.generator.IsEnabled() && needsGeneration {
		lengthBucket := model.LengthM
		if len(settings.LengthBuckets) > 0 {
			lengthBucket = settings.LengthBuckets[rand.Intn(len(settings.LengthBuckets))]
		}

		item, err = s.generator.GenerateItem(ctx, settings.Situations, lengthBucket, settings.CustomTopic)
		if err != nil {
			return nil, fmt.Errorf("failed to generate item: %w", err)
		}

		item.UserID = userID
		item.CustomTopic = settings.CustomTopic
		s.storeGeneratedItem(sessionID, item)
	}

	if item == nil {
		return nil, fmt.Errorf("no item available")
	}

	s.mu.RLock()
	currentIndex := session.CurrentIndex
	totalQuestions := session.TotalQuestions
	s.mu.RUnlock()

	return &model.PracticeQuestion{
		ItemID:         item.ItemID,
		Japanese:       item.Japanese,
		LengthBucket:   item.LengthBucket,
		Difficulty:     item.Difficulty,
		QuestionNumber: currentIndex + 1,
		TotalQuestions: totalQuestions,
	}, nil
}

// SubmitAnswer submits an answer and returns the result
func (s *PracticeService) SubmitAnswer(ctx context.Context, sessionID, itemID, userInput string, responseTimeMs int64) (*model.AnswerResult, error) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	userID := ""
	if ok {
		userID = session.UserID
	}
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// Get the item (check generated items first, then DB)
	item := s.getGeneratedItem(sessionID, itemID)
	isGenerated := item != nil
	if item == nil {
		var err error
		item, err = s.itemRepo.GetByID(ctx, userID, itemID)
		if err != nil {
			return nil, fmt.Errorf("failed to get item: %w", err)
		}
	}
	if item == nil {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}

	// Score the answer
	scoreResult := s.scorer.Score(userInput, item.Answers, item.Acceptable)

	// Update session with lock
	s.mu.Lock()
	session.CurrentIndex++
	session.AnsweredItemIDs = append(session.AnsweredItemIDs, itemID)
	if scoreResult.IsCorrect {
		session.CorrectCount++
	}
	s.mu.Unlock()

	// Save AI-generated item to DB for future review
	if isGenerated {
		item.CreatedAt = time.Now()
		if err := s.itemRepo.Create(ctx, item); err != nil {
			log.Printf("failed to save generated item: %v", err)
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

	result := &model.AnswerResult{
		IsCorrect:    scoreResult.IsCorrect,
		UserInput:    userInput,
		ModelAnswers: item.Answers,
		Acceptable:   item.Acceptable,
		MatchedWith:  scoreResult.MatchedWith,
	}

	// Generate coach feedback if AI is enabled
	if s.generator != nil && s.generator.IsEnabled() {
		feedback, err := s.generator.GenerateCoachFeedback(
			ctx,
			item.Japanese,
			userInput,
			item.Answers,
			item.Acceptable,
			scoreResult.IsCorrect,
		)
		if err != nil {
			log.Printf("failed to generate coach feedback: %v", err)
		} else {
			result.Feedback = feedback
		}
	}

	return result, nil
}

// GetSession retrieves a session by ID
func (s *PracticeService) GetSession(sessionID string) (*model.PracticeSession, error) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// GetHint returns a hint for a given item
func (s *PracticeService) GetHint(ctx context.Context, sessionID, itemID string, level int) (string, error) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	// Get the item (check generated items first, then DB)
	item := s.getGeneratedItem(sessionID, itemID)
	if item == nil {
		var err error
		item, err = s.itemRepo.GetByID(ctx, session.UserID, itemID)
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
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionID + ":" + item.ItemID
	s.generatedItems[key] = item
}

// getGeneratedItem retrieves a temporarily stored AI-generated item
func (s *PracticeService) getGeneratedItem(sessionID, itemID string) *model.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
