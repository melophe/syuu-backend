package scorer

import (
	"regexp"
	"strings"
	"unicode/utf8"
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

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return utf8.RuneCountInString(b)
	}
	if len(b) == 0 {
		return utf8.RuneCountInString(a)
	}

	aRunes := []rune(a)
	bRunes := []rune(b)

	lenA := len(aRunes)
	lenB := len(bRunes)

	// Create matrix
	matrix := make([][]int, lenA+1)
	for i := range matrix {
		matrix[i] = make([]int, lenB+1)
	}

	// Initialize first column
	for i := 0; i <= lenA; i++ {
		matrix[i][0] = i
	}

	// Initialize first row
	for j := 0; j <= lenB; j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 0
			if aRunes[i-1] != bRunes[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[lenA][lenB]
}

// similarity calculates similarity ratio (0.0 to 1.0)
func similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	// Use rune count (not byte length) for proper Unicode handling
	maxLen := max(utf8.RuneCountInString(a), utf8.RuneCountInString(b))
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshteinDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// Score checks if the user input matches any of the expected answers
// Uses fuzzy matching with 90% similarity threshold for minor typos
func (s *Scorer) Score(userInput string, modelAnswers []string, acceptable []string) Result {
	normalizedInput := s.Normalize(userInput)
	const similarityThreshold = 0.90

	// Check against model answers first (exact match)
	for _, answer := range modelAnswers {
		normalizedAnswer := s.Normalize(answer)
		if normalizedAnswer == normalizedInput {
			return Result{IsCorrect: true, MatchedWith: answer}
		}
	}

	// Check against acceptable alternatives (exact match)
	for _, answer := range acceptable {
		normalizedAnswer := s.Normalize(answer)
		if normalizedAnswer == normalizedInput {
			return Result{IsCorrect: true, MatchedWith: answer}
		}
	}

	// Fuzzy match against model answers
	for _, answer := range modelAnswers {
		normalizedAnswer := s.Normalize(answer)
		if similarity(normalizedAnswer, normalizedInput) >= similarityThreshold {
			return Result{IsCorrect: true, MatchedWith: answer}
		}
	}

	// Fuzzy match against acceptable alternatives
	for _, answer := range acceptable {
		normalizedAnswer := s.Normalize(answer)
		if similarity(normalizedAnswer, normalizedInput) >= similarityThreshold {
			return Result{IsCorrect: true, MatchedWith: answer}
		}
	}

	return Result{IsCorrect: false, MatchedWith: ""}
}
