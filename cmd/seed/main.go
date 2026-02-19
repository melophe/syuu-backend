package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/losts/syun-eng/backend/internal/config"
	"github.com/losts/syun-eng/backend/internal/model"
	"github.com/losts/syun-eng/backend/internal/repository"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	dbClient, err := repository.NewDynamoDBClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB client: %v", err)
	}

	itemRepo := repository.NewItemRepository(dbClient, cfg.ItemsTable)

	items := getSeedItems()

	fmt.Printf("Seeding %d items...\n", len(items))

	if err := itemRepo.BatchCreate(ctx, items); err != nil {
		log.Fatalf("Failed to seed items: %v", err)
	}

	fmt.Println("Seeding complete!")

	// Also write to JSON for reference
	writeToJSON(items)
}

func writeToJSON(items []*model.Item) {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal items: %v", err)
		return
	}

	if err := os.WriteFile("seed_data.json", data, 0644); err != nil {
		log.Printf("Failed to write JSON: %v", err)
		return
	}

	fmt.Println("Seed data written to seed_data.json")
}

func getSeedItems() []*model.Item {
	now := time.Now()

	return []*model.Item{
		// === Incident / Reporting (障害対応・報告) ===
		{
			ItemID:       "001",
			Japanese:     "本番環境でエラーが発生しています",
			Answers:      []string{"We're seeing errors in production"},
			Acceptable:   []string{"There are errors in production", "Errors are occurring in production", "We have errors in production"},
			Situations:   []string{"incident", "reporting"},
			Patterns:     []string{"reporting"},
			WordCount:    6,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "002",
			Japanese:     "サーバーがダウンしています",
			Answers:      []string{"The server is down"},
			Acceptable:   []string{"Server is down", "Our server is down"},
			Situations:   []string{"incident"},
			Patterns:     []string{"reporting"},
			WordCount:    4,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "003",
			Japanese:     "ユーザーからログインできないという報告がありました",
			Answers:      []string{"We received reports that users can't log in"},
			Acceptable:   []string{"Users are reporting they can't log in", "We got reports that users cannot log in"},
			Situations:   []string{"incident", "reporting"},
			Patterns:     []string{"reporting"},
			WordCount:    9,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "004",
			Japanese:     "原因を調査中です",
			Answers:      []string{"We're investigating the cause"},
			Acceptable:   []string{"We are investigating the cause", "Currently investigating the cause", "We're looking into the cause"},
			Situations:   []string{"incident"},
			Patterns:     []string{"mitigation"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "005",
			Japanese:     "問題を特定しました。修正をデプロイ中です",
			Answers:      []string{"We've identified the issue and are deploying a fix"},
			Acceptable:   []string{"We identified the issue and are deploying a fix", "The issue has been identified and we're deploying a fix"},
			Situations:   []string{"incident"},
			Patterns:     []string{"mitigation"},
			WordCount:    10,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},

		// === Deploy (デプロイ) ===
		{
			ItemID:       "006",
			Japanese:     "新しいバージョンをデプロイしてもいいですか",
			Answers:      []string{"Can I deploy the new version?"},
			Acceptable:   []string{"Is it okay to deploy the new version?", "May I deploy the new version?"},
			Situations:   []string{"deploy"},
			Patterns:     []string{"request"},
			WordCount:    6,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "007",
			Japanese:     "デプロイが完了しました",
			Answers:      []string{"The deployment is complete"},
			Acceptable:   []string{"Deployment complete", "The deploy is done", "Deployment finished"},
			Situations:   []string{"deploy"},
			Patterns:     []string{"reporting"},
			WordCount:    4,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "008",
			Japanese:     "デプロイ前に本番環境のバックアップを取る必要があります",
			Answers:      []string{"We need to back up the production environment before deploying"},
			Acceptable:   []string{"We should back up production before the deployment", "A production backup is needed before deployment"},
			Situations:   []string{"deploy"},
			Patterns:     []string{"recommendation"},
			WordCount:    10,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "009",
			Japanese:     "ロールバックの準備はできていますか",
			Answers:      []string{"Is the rollback ready?"},
			Acceptable:   []string{"Do we have a rollback ready?", "Is the rollback prepared?"},
			Situations:   []string{"deploy"},
			Patterns:     []string{"request"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},

		// === Design Review (設計レビュー) ===
		{
			ItemID:       "010",
			Japanese:     "この設計だとスケーラビリティに問題があります",
			Answers:      []string{"This design has scalability issues"},
			Acceptable:   []string{"There are scalability issues with this design", "This design won't scale well"},
			Situations:   []string{"review"},
			Patterns:     []string{"cause-effect"},
			WordCount:    6,
			LengthBucket: model.LengthS,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "011",
			Japanese:     "なぜこのアプローチを選んだのですか",
			Answers:      []string{"Why did you choose this approach?"},
			Acceptable:   []string{"What made you choose this approach?", "Why this approach?"},
			Situations:   []string{"review"},
			Patterns:     []string{"request"},
			WordCount:    6,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "012",
			Japanese:     "代替案を検討しましたか",
			Answers:      []string{"Did you consider any alternatives?"},
			Acceptable:   []string{"Have you considered alternatives?", "Were any alternatives considered?"},
			Situations:   []string{"review"},
			Patterns:     []string{"request"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "013",
			Japanese:     "パフォーマンスへの影響を考慮する必要があります",
			Answers:      []string{"We need to consider the performance impact"},
			Acceptable:   []string{"The performance impact needs to be considered", "We should consider the impact on performance"},
			Situations:   []string{"review"},
			Patterns:     []string{"recommendation"},
			WordCount:    8,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},

		// === Request (依頼・お願い) ===
		{
			ItemID:       "014",
			Japanese:     "このPRをレビューしてもらえますか",
			Answers:      []string{"Could you review this PR?"},
			Acceptable:   []string{"Can you review this PR?", "Would you mind reviewing this PR?"},
			Situations:   []string{"request"},
			Patterns:     []string{"request"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "015",
			Japanese:     "この件について詳しく説明していただけますか",
			Answers:      []string{"Could you explain this in more detail?"},
			Acceptable:   []string{"Can you give more details on this?", "Would you mind explaining this further?"},
			Situations:   []string{"request"},
			Patterns:     []string{"request"},
			WordCount:    7,
			LengthBucket: model.LengthM,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "016",
			Japanese:     "この機能の優先度を上げてもらえますか",
			Answers:      []string{"Could you prioritize this feature?"},
			Acceptable:   []string{"Can we prioritize this feature?", "Could you raise the priority of this feature?"},
			Situations:   []string{"request"},
			Patterns:     []string{"request"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   2,
			CreatedAt:    now,
		},

		// === Progress Report (進捗報告) ===
		{
			ItemID:       "017",
			Japanese:     "予定通り進んでいます",
			Answers:      []string{"We're on track"},
			Acceptable:   []string{"Everything is on schedule", "We're on schedule"},
			Situations:   []string{"progress"},
			Patterns:     []string{"reporting"},
			WordCount:    3,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "018",
			Japanese:     "予想より時間がかかっています",
			Answers:      []string{"It's taking longer than expected"},
			Acceptable:   []string{"This is taking longer than expected", "It's taking more time than expected"},
			Situations:   []string{"progress"},
			Patterns:     []string{"reporting"},
			WordCount:    6,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "019",
			Japanese:     "ブロッカーがあり、他チームの対応待ちです",
			Answers:      []string{"We have a blocker and are waiting for another team"},
			Acceptable:   []string{"There's a blocker and we're waiting on another team", "We're blocked and waiting for another team's response"},
			Situations:   []string{"progress"},
			Patterns:     []string{"reporting"},
			WordCount:    10,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "020",
			Japanese:     "今週中に完了する予定です",
			Answers:      []string{"We expect to complete it by the end of this week"},
			Acceptable:   []string{"It should be done by the end of this week", "We'll finish it by this week"},
			Situations:   []string{"progress"},
			Patterns:     []string{"reporting"},
			WordCount:    11,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},

		// === Casual / Small Talk (雑談) ===
		{
			ItemID:       "021",
			Japanese:     "週末はどうでしたか",
			Answers:      []string{"How was your weekend?"},
			Acceptable:   []string{"Did you have a good weekend?"},
			Situations:   []string{"casual"},
			Patterns:     []string{"request"},
			WordCount:    4,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "022",
			Japanese:     "最近忙しいですか",
			Answers:      []string{"Have you been busy lately?"},
			Acceptable:   []string{"Are you busy these days?", "Been busy lately?"},
			Situations:   []string{"casual"},
			Patterns:     []string{"request"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},

		// === Technical Discussion ===
		{
			ItemID:       "023",
			Japanese:     "このAPIはレート制限があります",
			Answers:      []string{"This API has rate limiting"},
			Acceptable:   []string{"This API is rate limited", "There's rate limiting on this API"},
			Situations:   []string{"technical"},
			Patterns:     []string{"constraints"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "024",
			Japanese:     "キャッシュを使ってパフォーマンスを改善できます",
			Answers:      []string{"We can improve performance by using caching"},
			Acceptable:   []string{"Caching can improve the performance", "We can use caching to improve performance"},
			Situations:   []string{"technical", "review"},
			Patterns:     []string{"recommendation"},
			WordCount:    8,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "025",
			Japanese:     "このクエリはインデックスが効いていません",
			Answers:      []string{"This query isn't using the index"},
			Acceptable:   []string{"This query doesn't use the index", "The index isn't being used by this query"},
			Situations:   []string{"technical"},
			Patterns:     []string{"cause-effect"},
			WordCount:    7,
			LengthBucket: model.LengthM,
			Difficulty:   3,
			CreatedAt:    now,
		},

		// === Meeting / Discussion ===
		{
			ItemID:       "026",
			Japanese:     "この件について話し合いましょう",
			Answers:      []string{"Let's discuss this"},
			Acceptable:   []string{"Let's talk about this", "Can we discuss this?"},
			Situations:   []string{"meeting"},
			Patterns:     []string{"request"},
			WordCount:    3,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "027",
			Japanese:     "ミーティングの時間を変更できますか",
			Answers:      []string{"Can we reschedule the meeting?"},
			Acceptable:   []string{"Could we change the meeting time?", "Is it possible to reschedule the meeting?"},
			Situations:   []string{"meeting", "request"},
			Patterns:     []string{"request"},
			WordCount:    5,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},

		// === Longer / Complex Sentences ===
		{
			ItemID:       "028",
			Japanese:     "この変更を本番にデプロイする前に、ステージング環境でテストする必要があります",
			Answers:      []string{"We need to test this change in the staging environment before deploying it to production"},
			Acceptable:   []string{"This change needs to be tested in staging before going to production"},
			Situations:   []string{"deploy", "review"},
			Patterns:     []string{"recommendation"},
			WordCount:    16,
			LengthBucket: model.LengthL,
			Difficulty:   3,
			CreatedAt:    now,
		},
		{
			ItemID:       "029",
			Japanese:     "データベースのマイグレーションは、トラフィックの少ない時間帯に実行することをお勧めします",
			Answers:      []string{"I recommend running the database migration during low traffic hours"},
			Acceptable:   []string{"The database migration should be run during low traffic hours", "We should run the database migration when traffic is low"},
			Situations:   []string{"deploy", "technical"},
			Patterns:     []string{"recommendation"},
			WordCount:    12,
			LengthBucket: model.LengthL,
			Difficulty:   3,
			CreatedAt:    now,
		},
		{
			ItemID:       "030",
			Japanese:     "この機能を実装するには、まずAPIの仕様を決める必要があります",
			Answers:      []string{"To implement this feature, we first need to define the API specification"},
			Acceptable:   []string{"Before implementing this feature, we need to define the API spec"},
			Situations:   []string{"technical", "progress"},
			Patterns:     []string{"cause-effect"},
			WordCount:    13,
			LengthBucket: model.LengthL,
			Difficulty:   3,
			CreatedAt:    now,
		},

		// === Additional items for variety ===
		{
			ItemID:       "031",
			Japanese:     "テストは全て通りました",
			Answers:      []string{"All tests passed"},
			Acceptable:   []string{"All the tests passed", "Tests are all passing"},
			Situations:   []string{"deploy", "progress"},
			Patterns:     []string{"reporting"},
			WordCount:    3,
			LengthBucket: model.LengthS,
			Difficulty:   1,
			CreatedAt:    now,
		},
		{
			ItemID:       "032",
			Japanese:     "コードレビューのコメントに対応しました",
			Answers:      []string{"I've addressed the code review comments"},
			Acceptable:   []string{"I addressed the review comments", "Code review comments have been addressed"},
			Situations:   []string{"review"},
			Patterns:     []string{"reporting"},
			WordCount:    6,
			LengthBucket: model.LengthS,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "033",
			Japanese:     "この問題の根本原因は何ですか",
			Answers:      []string{"What's the root cause of this issue?"},
			Acceptable:   []string{"What is the root cause?", "What caused this issue?"},
			Situations:   []string{"incident", "technical"},
			Patterns:     []string{"request"},
			WordCount:    7,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},
		{
			ItemID:       "034",
			Japanese:     "リソースの制約があるため、優先順位を決める必要があります",
			Answers:      []string{"Due to resource constraints, we need to prioritize"},
			Acceptable:   []string{"We need to prioritize because of resource constraints", "Given the resource constraints, we need to set priorities"},
			Situations:   []string{"progress", "meeting"},
			Patterns:     []string{"cause-effect"},
			WordCount:    9,
			LengthBucket: model.LengthM,
			Difficulty:   3,
			CreatedAt:    now,
		},
		{
			ItemID:       "035",
			Japanese:     "この機能はMVPに必要ですか",
			Answers:      []string{"Is this feature necessary for the MVP?"},
			Acceptable:   []string{"Do we need this for the MVP?", "Is this required for the MVP?"},
			Situations:   []string{"review", "meeting"},
			Patterns:     []string{"request"},
			WordCount:    8,
			LengthBucket: model.LengthM,
			Difficulty:   2,
			CreatedAt:    now,
		},
	}
}
