package scorer

import (
	"regexp"
	"strings"
)

// Scorer handles answer scoring
type Scorer struct {
	contractions map[string]string
}

// New creates a new Scorer instance
func New() *Scorer {
	return &Scorer{
		contractions: map[string]string{
			"i'm":       "i am",
			"you're":    "you are",
			"he's":      "he is",
			"she's":     "she is",
			"it's":      "it is",
			"we're":     "we are",
			"they're":   "they are",
			"i've":      "i have",
			"you've":    "you have",
			"we've":     "we have",
			"they've":   "they have",
			"i'd":       "i would",
			"you'd":     "you would",
			"he'd":      "he would",
			"she'd":     "she would",
			"we'd":      "we would",
			"they'd":    "they would",
			"i'll":      "i will",
			"you'll":    "you will",
			"he'll":     "he will",
			"she'll":    "she will",
			"we'll":     "we will",
			"they'll":   "they will",
			"isn't":     "is not",
			"aren't":    "are not",
			"wasn't":    "was not",
			"weren't":   "were not",
			"haven't":   "have not",
			"hasn't":    "has not",
			"hadn't":    "had not",
			"won't":     "will not",
			"wouldn't":  "would not",
			"don't":     "do not",
			"doesn't":   "does not",
			"didn't":    "did not",
			"can't":     "cannot",
			"couldn't":  "could not",
			"shouldn't": "should not",
			"mightn't":  "might not",
			"mustn't":   "must not",
			"let's":     "let us",
			"that's":    "that is",
			"who's":     "who is",
			"what's":    "what is",
			"here's":    "here is",
			"there's":   "there is",
			"where's":   "where is",
			"when's":    "when is",
			"how's":     "how is",
			"what're":   "what are",
			"where're":  "where are",
			"who're":    "who are",
		},
	}
}

// Normalize normalizes a string for comparison
func (s *Scorer) Normalize(input string) string {
	// Convert to lowercase
	result := strings.ToLower(input)

	// Trim whitespace
	result = strings.TrimSpace(result)

	// Normalize multiple spaces to single space
	spaceRegex := regexp.MustCompile(`\s+`)
	result = spaceRegex.ReplaceAllString(result, " ")

	// Expand contractions BEFORE removing punctuation (apostrophes matter here)
	words := strings.Split(result, " ")
	for i, word := range words {
		if expanded, ok := s.contractions[word]; ok {
			words[i] = expanded
		}
	}
	result = strings.Join(words, " ")

	// Remove punctuation AFTER expanding contractions
	punctuationRegex := regexp.MustCompile(`[.,!?;:'"()-]`)
	result = punctuationRegex.ReplaceAllString(result, "")

	// Normalize spaces again after punctuation removal
	result = spaceRegex.ReplaceAllString(result, " ")

	// Final trim
	result = strings.TrimSpace(result)

	return result
}

// Result represents the scoring result
type Result struct {
	IsCorrect   bool
	MatchedWith string // The answer that was matched (empty if incorrect)
}

// Score checks if the user input matches any of the expected answers
func (s *Scorer) Score(userInput string, modelAnswers []string, acceptable []string) Result {
	normalizedInput := s.Normalize(userInput)

	// Check against model answers first
	for _, answer := range modelAnswers {
		if s.Normalize(answer) == normalizedInput {
			return Result{IsCorrect: true, MatchedWith: answer}
		}
	}

	// Check against acceptable alternatives
	for _, answer := range acceptable {
		if s.Normalize(answer) == normalizedInput {
			return Result{IsCorrect: true, MatchedWith: answer}
		}
	}

	return Result{IsCorrect: false, MatchedWith: ""}
}

// ScoreWithFuzzy performs fuzzy matching (for future expansion)
// Currently just wraps Score, but can be extended with Levenshtein distance
func (s *Scorer) ScoreWithFuzzy(userInput string, modelAnswers []string, acceptable []string, threshold float64) Result {
	// For MVP, just use exact matching
	return s.Score(userInput, modelAnswers, acceptable)
}
