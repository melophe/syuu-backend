package model

// PracticeSettings represents the settings for a practice session
type PracticeSettings struct {
	Situations     []string       `json:"situations"`      // Selected situation tags
	LengthBuckets  []LengthBucket `json:"length_buckets"`  // Selected length buckets
	QuestionCount  int            `json:"question_count"`  // Number of questions (0 = infinite)
	ReviewPriority bool           `json:"review_priority"` // Prioritize due SRS items
	CustomTopic    string         `json:"custom_topic"`    // Custom topic for AI generation
}

// PracticeSession represents an active practice session
type PracticeSession struct {
	SessionID        string           `json:"session_id"`
	UserID           string           `json:"user_id"`
	Settings         PracticeSettings `json:"settings"`
	CurrentIndex     int              `json:"current_index"`
	TotalQuestions   int              `json:"total_questions"`
	CorrectCount     int              `json:"correct_count"`
	ItemIDs          []string         `json:"item_ids"`           // Ordered list of item IDs for this session
	AnsweredItemIDs  []string         `json:"answered_item_ids"`  // Items already answered
}

// PracticeQuestion represents a question presented to the user
type PracticeQuestion struct {
	ItemID         string       `json:"item_id"`
	Japanese       string       `json:"japanese"`
	LengthBucket   LengthBucket `json:"length_bucket"`
	Difficulty     int          `json:"difficulty"`
	QuestionNumber int          `json:"question_number"`
	TotalQuestions int          `json:"total_questions"`
}

// AnswerResult represents the result of checking an answer
type AnswerResult struct {
	IsCorrect    bool     `json:"is_correct"`
	UserInput    string   `json:"user_input"`
	ModelAnswers []string `json:"model_answers"`
	Acceptable   []string `json:"acceptable"`
	MatchedWith  string   `json:"matched_with,omitempty"` // Which answer it matched (if correct)
	Explanation  string   `json:"explanation,omitempty"`  // Brief explanation if needed
}
