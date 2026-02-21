package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/losts/syun-eng/backend/internal/config"
	"github.com/losts/syun-eng/backend/internal/handler"
	"github.com/losts/syun-eng/backend/internal/middleware"
	"github.com/losts/syun-eng/backend/internal/repository"
	"github.com/losts/syun-eng/backend/internal/service"
)

func main() {
	// Load .env file (ignore error if not exists)
	_ = godotenv.Load()

	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create DynamoDB client
	ctx := context.Background()
	dbClient, err := repository.NewDynamoDBClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB client: %v", err)
	}

	// Create repositories
	itemRepo := repository.NewItemRepository(dbClient, cfg.ItemsTable)
	userRepo := repository.NewUserRepository(dbClient, cfg.UsersTable)
	srsRepo := repository.NewSRSRepository(dbClient, cfg.SRSTable)
	answerRepo := repository.NewAnswerRepository(dbClient, cfg.AnswersTable)

	// Create services
	authService := service.NewAuthService(userRepo, cfg)
	generatorService := service.NewGeneratorService(cfg)
	practiceService := service.NewPracticeService(itemRepo, srsRepo, answerRepo, generatorService)
	statsService := service.NewStatsService(answerRepo, srsRepo)

	// Create handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	practiceHandler := handler.NewPracticeHandler(practiceService)
	statsHandler := handler.NewStatsHandler(statsService)

	// Create router
	r := gin.Default()

	// Apply CORS middleware
	r.Use(middleware.CORSMiddleware(cfg.FrontendURL))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes (public)
	auth := r.Group("/auth")
	{
		auth.POST("/google/callback", authHandler.GoogleCallback)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// Protected routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(authService, cfg))
	{
		// User
		api.GET("/me", authHandler.Me)

		// Practice
		practice := api.Group("/practice")
		{
			practice.POST("/start", practiceHandler.StartSession)
			practice.GET("/:session_id", practiceHandler.GetSession)
			practice.GET("/:session_id/next", practiceHandler.GetNextQuestion)
			practice.POST("/:session_id/answer", practiceHandler.SubmitAnswer)
			practice.DELETE("/:session_id", practiceHandler.EndSession)
		}

		// Stats
		stats := api.Group("/stats")
		{
			stats.GET("/summary", statsHandler.GetSummary)
			stats.GET("/weakness", statsHandler.GetWeaknesses)
		}
	}

	// Create server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
