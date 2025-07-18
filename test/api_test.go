package test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yourname/sleeptracker/internal"
	api "github.com/yourname/sleeptracker/internal/api"
	"github.com/yourname/sleeptracker/internal/auth"
	"github.com/yourname/sleeptracker/internal/config"
	"github.com/yourname/sleeptracker/internal/storage"
	"go.uber.org/zap"
)

type TestApp struct {
	logger    internal.Logger
	sleepRepo storage.SleepLogRepository
	goalRepo  storage.GoalRepository
}

func (a *TestApp) Logger() internal.Logger               { return a.logger }
func (a *TestApp) SleepRepo() storage.SleepLogRepository { return a.sleepRepo }
func (a *TestApp) GoalRepo() storage.GoalRepository      { return a.goalRepo }

func setupRouterAndStorage(t *testing.T) (*gin.Engine, *TestApp) {
	gin.SetMode(gin.TestMode)
	testDir := "testdata"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		_ = os.MkdirAll(testDir, 0755)
	}
	sleepFile := testDir + "/test_sleep_logs.json"
	goalsFile := testDir + "/test_goals.json"
	os.Remove(sleepFile)
	os.Remove(goalsFile)
	sleepRepo, goalRepo, err := storage.NewFileRepositories(sleepFile, goalsFile, internal.NewZapLogger(zap.NewNop().Sugar()))
	assert.NoError(t, err)
	logger := internal.NewZapLogger(zap.NewNop().Sugar())
	app := &TestApp{
		logger:    logger,
		sleepRepo: sleepRepo,
		goalRepo:  goalRepo,
	}
	cfg := &config.Config{Env: "development"}
	r := gin.Default()
	r.Use(auth.AuthMiddleware(auth.NewLocalAuthProvider("MOCK-TOKEN", logger), cfg))
	r.POST("/sleep", api.PostSleep(app))
	r.GET("/sleep", api.GetSleep(app))
	r.GET("/sleep/stats", api.GetSleepStats(app))
	r.GET("/sleep/recommendations", api.GetSleepRecommendations(app))
	r.POST("/api/goals", api.PostGoal(app))
	r.GET("/api/goals/progress", api.GetGoalProgress(app))
	return r, app
}

func TestPostGoal_ValidAndInvalid(t *testing.T) {
	r, _ := setupRouterAndStorage(t)
	ts := httptest.NewRecorder()
	// Valid
	body := `{"type":"duration","value":"7h"}`
	req, _ := http.NewRequest("POST", "/api/goals", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 201, ts.Code)
	// Invalid: missing value
	ts = httptest.NewRecorder()
	body = `{"type":"duration"}`
	req, _ = http.NewRequest("POST", "/api/goals", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
	// Invalid: unsupported type
	ts = httptest.NewRecorder()
	body = `{"type":"banana","value":"7h"}`
	req, _ = http.NewRequest("POST", "/api/goals", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
}

func TestPostSleep_ValidAndInvalid(t *testing.T) {
	r, app := setupRouterAndStorage(t)
	ts := httptest.NewRecorder()
	// Valid
	start := time.Now().Add(-8 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	body := `{"start_time":"` + start + `","end_time":"` + end + `","quality":7}`
	app.logger.Infof("TestPostSleep_ValidAndInvalid valid request body: %s", body)
	req, _ := http.NewRequest("POST", "/sleep", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	app.logger.Infof("TestPostSleep_ValidAndInvalid valid response: %s", ts.Body.String())
	assert.Equal(t, 201, ts.Code)
	// Invalid: quality out of range
	ts = httptest.NewRecorder()
	body = `{"start_time":"` + start + `","end_time":"` + end + `","quality":99}`
	req, _ = http.NewRequest("POST", "/sleep", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
	// Invalid: missing start_time
	ts = httptest.NewRecorder()
	body = `{"end_time":"` + end + `","quality":7}`
	req, _ = http.NewRequest("POST", "/sleep", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
}

func TestGetGoalProgress_NoGoal(t *testing.T) {
	r, _ := setupRouterAndStorage(t)
	ts := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/goals/progress", nil)
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 404, ts.Code)
}

func TestSleepAPI(t *testing.T) {
	r, app := setupRouterAndStorage(t)
	w := httptest.NewRecorder()
	jsonBody := `{"start_time":"2025-07-16T22:00:00Z","end_time":"2025-07-17T06:00:00Z","quality":8,"reason":"Felt rested","interruptions":["bathroom"]}`
	app.logger.Infof("TestSleepAPI request body: %s", jsonBody)
	req, _ := http.NewRequest("POST", "/sleep", strings.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	app.logger.Infof("TestSleepAPI response: %s", w.Body.String())
	assert.Equal(t, 201, w.Code)
}

func TestSleepAuthFail(t *testing.T) {
	r, _ := setupRouterAndStorage(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sleep",
		strings.NewReader(`{"start_time":"2025-07-16T22:00:00Z","end_time":"2025-07-17T06:00:00Z","quality":8}`))
	req.Header.Set("Authorization", "Bearer WRONG-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)
}
