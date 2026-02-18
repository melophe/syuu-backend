package scorer

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "remove punctuation",
			input:    "Hello, World!",
			expected: "hello world",
		},
		{
			name:     "trim whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "normalize multiple spaces",
			input:    "hello   world",
			expected: "hello world",
		},
		{
			name:     "expand contractions - I'm",
			input:    "I'm working",
			expected: "i am working",
		},
		{
			name:     "expand contractions - We're",
			input:    "We're seeing errors",
			expected: "we are seeing errors",
		},
		{
			name:     "expand contractions - don't",
			input:    "I don't know",
			expected: "i do not know",
		},
		{
			name:     "expand contractions - can't",
			input:    "I can't do it",
			expected: "i cannot do it",
		},
		{
			name:     "complex sentence",
			input:    "We're seeing errors in production, and we can't fix it!",
			expected: "we are seeing errors in production and we cannot fix it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestScore(t *testing.T) {
	s := New()

	tests := []struct {
		name         string
		userInput    string
		modelAnswers []string
		acceptable   []string
		wantCorrect  bool
		wantMatched  string
	}{
		{
			name:         "exact match",
			userInput:    "We're seeing errors in production",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{},
			wantCorrect:  true,
			wantMatched:  "We're seeing errors in production",
		},
		{
			name:         "case insensitive match",
			userInput:    "we're seeing errors in production",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{},
			wantCorrect:  true,
			wantMatched:  "We're seeing errors in production",
		},
		{
			name:         "punctuation insensitive",
			userInput:    "We're seeing errors in production!",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{},
			wantCorrect:  true,
			wantMatched:  "We're seeing errors in production",
		},
		{
			name:         "contraction expansion match",
			userInput:    "We are seeing errors in production",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{},
			wantCorrect:  true,
			wantMatched:  "We're seeing errors in production",
		},
		{
			name:         "match acceptable answer",
			userInput:    "There are errors in production",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{"There are errors in production"},
			wantCorrect:  true,
			wantMatched:  "There are errors in production",
		},
		{
			name:         "no match",
			userInput:    "The server is down",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{"There are errors in production"},
			wantCorrect:  false,
			wantMatched:  "",
		},
		{
			name:         "extra whitespace",
			userInput:    "  We're  seeing  errors  in  production  ",
			modelAnswers: []string{"We're seeing errors in production"},
			acceptable:   []string{},
			wantCorrect:  true,
			wantMatched:  "We're seeing errors in production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Score(tt.userInput, tt.modelAnswers, tt.acceptable)
			if result.IsCorrect != tt.wantCorrect {
				t.Errorf("Score() IsCorrect = %v, want %v", result.IsCorrect, tt.wantCorrect)
			}
			if result.MatchedWith != tt.wantMatched {
				t.Errorf("Score() MatchedWith = %q, want %q", result.MatchedWith, tt.wantMatched)
			}
		})
	}
}
