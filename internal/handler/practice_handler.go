package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/losts/syun-eng/backend/internal/model"
	"github.com/losts/syun-eng/backend/internal/service"
)

// PracticeHandler handles practice-related HTTP requests
type PracticeHandler struct {
	practiceService *service.PracticeService
}

// NewPracticeHandler creates a new PracticeHandler
func NewPracticeHandler(practiceService *service.PracticeService) *PracticeHandler {
	return &PracticeHandler{
		practiceService: practiceService,
	}
}

// StartSessionRequest represents the request to start a session
type StartSessionRequest struct {
	Situations     []string             `json:"situations"`
	LengthBuckets  []model.LengthBucket `json:"length_buckets"`
	QuestionCount  int                  `json:"question_count"`
	ReviewPriority bool                 `json:"review_priority"`
	CustomTopic    string               `json:"custom_topic"`
}

// StartSessionResponse represents the response after starting a session
type StartSessionResponse struct {
	SessionID      string                 `json:"session_id"`
	TotalQuestions int                    `json:"total_questions"`
	FirstQuestion  *model.PracticeQuestion `json:"first_question,omitempty"`
}

// StartSession starts a new practice session
func (h *PracticeHandler) StartSession(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings := model.PracticeSettings{
		Situations:     req.Situations,
		LengthBuckets:  req.LengthBuckets,
		QuestionCount:  req.QuestionCount,
		ReviewPriority: req.ReviewPriority,
		CustomTopic:    req.CustomTopic,
	}

	session, err := h.practiceService.StartSession(c.Request.Context(), userID, settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get first question
	question, err := h.practiceService.GetNextQuestion(c.Request.Context(), session.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, StartSessionResponse{
		SessionID:      session.SessionID,
		TotalQuestions: session.TotalQuestions,
		FirstQuestion:  question,
	})
}

// GetNextQuestion gets the next question in the session
func (h *PracticeHandler) GetNextQuestion(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	// Verify session ownership
	session, err := h.practiceService.GetSession(sessionID)
	if err != nil || session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	question, err := h.practiceService.GetNextQuestion(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if question == nil {
		// Session complete
		session, err := h.practiceService.GetSession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"complete":       true,
			"total":          session.TotalQuestions,
			"correct":        session.CorrectCount,
			"accuracy_rate":  float64(session.CorrectCount) / float64(session.TotalQuestions) * 100,
		})
		return
	}

	c.JSON(http.StatusOK, question)
}

// SubmitAnswerRequest represents the request to submit an answer
type SubmitAnswerRequest struct {
	ItemID         string `json:"item_id" binding:"required"`
	UserInput      string `json:"user_input" binding:"required"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// SubmitAnswer submits an answer and returns the result
func (h *PracticeHandler) SubmitAnswer(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	// Verify session ownership
	session, err := h.practiceService.GetSession(sessionID)
	if err != nil || session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	var req SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.practiceService.SubmitAnswer(
		c.Request.Context(),
		sessionID,
		req.ItemID,
		req.UserInput,
		req.ResponseTimeMs,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSession gets session information
func (h *PracticeHandler) GetSession(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	session, err := h.practiceService.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Verify session ownership
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":       session.SessionID,
		"current_index":    session.CurrentIndex,
		"total_questions":  session.TotalQuestions,
		"correct_count":    session.CorrectCount,
		"settings":         session.Settings,
	})
}

// GetHintRequest represents the request to get a hint
type GetHintRequest struct {
	ItemID string `json:"item_id" binding:"required"`
	Level  int    `json:"level"`
}

// GetHint returns a hint for the current question
func (h *PracticeHandler) GetHint(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	// Verify session ownership
	session, err := h.practiceService.GetSession(sessionID)
	if err != nil || session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	var req GetHintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to level 1 if not specified
	level := req.Level
	if level < 1 || level > 3 {
		level = 1
	}

	hint, err := h.practiceService.GetHint(c.Request.Context(), sessionID, req.ItemID, level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"hint": hint, "level": level})
}

// EndSession ends a practice session
func (h *PracticeHandler) EndSession(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	// Verify session ownership
	session, err := h.practiceService.GetSession(sessionID)
	if err != nil || session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	h.practiceService.EndSession(sessionID)
	c.JSON(http.StatusOK, gin.H{"message": "session ended"})
}
