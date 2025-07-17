package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal"
	api "github.com/yourname/sleeptracker/internal/api"
)

func main() {
	dataDir := "data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		_ = os.Mkdir(dataDir, 0755)
	}
	usersFile := dataDir + "/users.json"
	sleepFile := dataDir + "/sleep_logs.json"
	goalsFile := dataDir + "/goals.json"
	// Create default user if not exists
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		f, _ := os.Create(usersFile)
		_ = f.Close()
		os.WriteFile(usersFile, []byte(`[{"id":"u1","token":"MOCK-TOKEN","name":"Demo User"}]`), 0644)
	}
	storage, err := internal.NewStorage(usersFile, sleepFile, goalsFile)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	r := gin.Default()

	// Serve the OpenAPI spec locally
	r.GET("/swagger.yaml", func(c *gin.Context) {
		c.File("swagger.yaml")
	})

	// Serve local Swagger UI static files
	r.Static("/swagger", "./swagger-ui")

	// Protected routes
	r.Use(api.AuthMiddleware(storage))
	r.POST("/sleep", api.PostSleep(storage))
	r.GET("/sleep", api.GetSleep(storage))
	r.GET("/sleep/stats", api.GetSleepStats(storage))
	r.GET("/sleep/recommendations", api.GetSleepRecommendations())
	r.POST("/api/goals", api.PostGoal(storage))
	r.GET("/api/goals/progress", api.GetGoalProgress(storage))

	go func() {
		log.Println("Server running on :8088")
		err := r.Run(":8088")
		if err != nil {
			log.Fatalf("failed to start server: %v", err)
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
