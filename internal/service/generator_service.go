package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/losts/syun-eng/backend/internal/config"
	"github.com/losts/syun-eng/backend/internal/model"
)

// GeneratorService generates practice items using Claude API
type GeneratorService struct {
	client  anthropic.Client
	enabled bool
}

// NewGeneratorService creates a new GeneratorService
func NewGeneratorService(cfg *config.Config) *GeneratorService {
	if cfg.AnthropicAPIKey == "" {
		return &GeneratorService{enabled: false}
	}

	client := anthropic.NewClient(option.WithAPIKey(cfg.AnthropicAPIKey))
	return &GeneratorService{
		client:  client,
		enabled: true,
	}
}

// IsEnabled returns whether the generator is enabled
func (g *GeneratorService) IsEnabled() bool {
	return g.enabled
}

// GeneratedItem represents the JSON structure from Claude
type GeneratedItem struct {
	Japanese   string   `json:"japanese"`
	Answers    []string `json:"answers"`
	Acceptable []string `json:"acceptable"`
	Difficulty int      `json:"difficulty"`
}

// GenerateItem generates a new practice item using Claude API
func (g *GeneratorService) GenerateItem(ctx context.Context, situations []string, lengthBucket model.LengthBucket, customTopic string) (*model.Item, error) {
	if !g.enabled {
		return nil, fmt.Errorf("generator service is not enabled (missing ANTHROPIC_API_KEY)")
	}

	lengthDesc := getLengthDescription(lengthBucket)
	situationList := "general"
	if len(situations) > 0 {
		situationList = strings.Join(situations, ", ")
	}

	topicCondition := ""
	if customTopic != "" {
		// Limit topic length to prevent token waste
		topic := customTopic
		if len(topic) > 100 {
			topic = topic[:100]
		}
		// Sanitize to prevent prompt injection
		topic = strings.ReplaceAll(topic, "」", "")
		topic = strings.ReplaceAll(topic, "「", "")
		topicCondition = fmt.Sprintf("\n- お題/トピック: 「%s」（この内容に関連したフレーズを生成）", topic)
	}

	prompt := fmt.Sprintf(`ITエンジニアが仕事で使う日本語フレーズとその英訳を1つ生成してください。

条件:
- シチュエーション: %s
- 文の長さ: %s%s

以下のJSON形式のみで出力してください（説明不要）:
{
  "japanese": "日本語の文",
  "answers": ["模範となる英訳を1-2個"],
  "acceptable": ["許容される別の英訳を1-2個"],
  "difficulty": 1から5の難易度
}`, situationList, lengthDesc, topicCondition)

	message, err := g.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 500,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call Claude API: %w", err)
	}

	// Extract text from response
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	// Parse JSON from response
	var generated GeneratedItem
	if err := json.Unmarshal([]byte(responseText), &generated); err != nil {
		// Try to extract JSON from response
		start := strings.Index(responseText, "{")
		end := strings.LastIndex(responseText, "}")
		if start >= 0 && end > start {
			jsonStr := responseText[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &generated); err != nil {
				return nil, fmt.Errorf("failed to parse generated item: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse generated item: %w", err)
		}
	}

	// Calculate word count from first answer
	wordCount := 0
	if len(generated.Answers) > 0 {
		wordCount = len(strings.Fields(generated.Answers[0]))
	}

	item := &model.Item{
		ItemID:       "gen-" + uuid.New().String()[:8],
		Japanese:     generated.Japanese,
		Answers:      generated.Answers,
		Acceptable:   generated.Acceptable,
		Situations:   situations,
		Patterns:     []string{},
		WordCount:    wordCount,
		LengthBucket: model.GetLengthBucket(wordCount),
		Difficulty:   generated.Difficulty,
		CreatedAt:    time.Now(),
	}

	return item, nil
}

// GenerateHint generates a hint for a given item
func (g *GeneratorService) GenerateHint(item *model.Item, level int) string {
	if len(item.Answers) == 0 {
		return "ヒントがありません"
	}

	answer := item.Answers[0]
	words := strings.Fields(answer)

	switch level {
	case 1:
		// Level 1: Show word count
		return fmt.Sprintf("%d語の文です", len(words))
	case 2:
		// Level 2: Show first word
		if len(words) > 0 {
			return fmt.Sprintf("最初の単語: %s ...", words[0])
		}
		return "ヒントがありません"
	case 3:
		// Level 3: Show first few words
		showCount := len(words) / 3
		if showCount < 2 {
			showCount = 2
		}
		if showCount > len(words) {
			showCount = len(words)
		}
		return strings.Join(words[:showCount], " ") + " ..."
	default:
		return fmt.Sprintf("%d語の文です", len(words))
	}
}

func getLengthDescription(bucket model.LengthBucket) string {
	switch bucket {
	case model.LengthS:
		return "S (短め: 6語以下)"
	case model.LengthM:
		return "M (中程度: 7-12語)"
	case model.LengthL:
		return "L (長め: 13-20語)"
	case model.LengthXL:
		return "XL (とても長い: 21語以上)"
	default:
		return "M (中程度: 7-12語)"
	}
}
