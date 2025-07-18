package main

import (
	"os/exec"
	"runtime"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal"
	api "github.com/yourname/sleeptracker/internal/api"
	"github.com/yourname/sleeptracker/internal/auth"
	"github.com/yourname/sleeptracker/internal/config"
	"github.com/yourname/sleeptracker/internal/storage"
	"go.uber.org/zap"
)

// App is the DI container for the application
type App struct {
	Config    *config.Config
	logger    internal.Logger
	sleepRepo storage.SleepLogRepository
	goalRepo  storage.GoalRepository
}

func (a *App) Logger() internal.Logger               { return a.logger }
func (a *App) SleepRepo() storage.SleepLogRepository { return a.sleepRepo }
func (a *App) GoalRepo() storage.GoalRepository      { return a.goalRepo }

func main() {
	cfg := config.Load()

	zapLogger, err := zap.NewProduction()
	if cfg.Env == "development" {
		zapLogger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	sugar := zapLogger.Sugar()
	defer zapLogger.Sync()
	logger := internal.NewZapLogger(sugar)

	var (
		sleepRepo storage.SleepLogRepository
		goalRepo  storage.GoalRepository
	)

	switch cfg.DBType {
	case "file":
		sleepRepo, goalRepo, err = storage.NewFileRepositories(cfg.FileSleep, cfg.FileGoals, logger)
		if err != nil {
			logger.Fatalf("failed to initialize repositories: %v", err)
		}
	case "postgres":
		if cfg.DBDSN == "" {
			logger.Fatalf("POSTGRES_DSN env var required for postgres backend")
		}
		// Make sure to run migrations/001_init.sql before starting the app
		sleepRepo, goalRepo, err = storage.NewPostgresRepositories(cfg.DBDSN, logger)
		if err != nil {
			logger.Fatalf("failed to initialize postgres repositories: %v", err)
		}
	default:
		logger.Fatalf("unsupported STORAGE_BACKEND: %s", cfg.DBType)
	}

	app := &App{
		Config:    cfg,
		logger:    logger,
		sleepRepo: sleepRepo,
		goalRepo:  goalRepo,
	}

	r := gin.Default()

	r.Use(api.RequestIDMiddleware())
	// Serve the OpenAPI spec locally
	r.GET("/swagger.yaml", func(c *gin.Context) {
		c.File("swagger.yaml")
	})

	// Serve local Swagger UI static files
	r.Static("/swagger", "./swagger-ui")

	// Protected routes
	var authProvider auth.Provider
	if cfg.Env == "development" {
		authProvider = auth.NewLocalAuthProvider("MOCK-TOKEN", logger)
	} else {
		authProvider = auth.NewRemoteAuthProvider(cfg.DBDSN, logger)
	}
	r.Use(auth.AuthMiddleware(authProvider, cfg))
	r.POST("/sleep", api.PostSleep(app))
	r.GET("/sleep", api.GetSleep(app))
	r.GET("/sleep/stats", api.GetSleepStats(app))
	r.GET("/sleep/recommendations", api.GetSleepRecommendations(app))
	r.POST("/api/goals", api.PostGoal(app))
	r.GET("/api/goals/progress", api.GetGoalProgress(app))

	go func() {
		app.Logger().Infof("Server running on :8088")
		err := r.Run(":8088")
		if err != nil {
			app.Logger().Fatalf("failed to start server: %v", err)
		}
	}()

	// Poll until Swagger UI is ready
	url := "http://localhost:8088/swagger/"
	for i := 0; i < 30; i++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Open browser
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()

	// Block main goroutine
	select {}
}
