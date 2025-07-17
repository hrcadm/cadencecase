package test

import (
	"io/ioutil"
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
)

func setupRouterAndStorage(t *testing.T) (*gin.Engine, *internal.Storage, string) {
	testDir := "testdata"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		_ = os.MkdirAll(testDir, 0755)
	}
	usersFile := testDir + "/test_users.json"
	sleepFile := testDir + "/test_sleep_logs.json"
	goalsFile := testDir + "/test_goals.json"
	os.Remove(usersFile)
	os.Remove(sleepFile)
	os.Remove(goalsFile)
	os.WriteFile(usersFile, []byte(`[{"id":"u1","token":"MOCK-TOKEN","name":"Test User"}]`), 0644)
	storage, err := internal.NewStorage(usersFile, sleepFile, goalsFile)
	assert.NoError(t, err)
	r := gin.Default()
	r.Use(api.AuthMiddleware(storage))
	r.POST("/sleep", api.PostSleep(storage))
	r.GET("/sleep", api.GetSleep(storage))
	r.GET("/sleep/stats", api.GetSleepStats(storage))
	r.GET("/sleep/recommendations", api.GetSleepRecommendations())
	r.POST("/api/goals", api.PostGoal(storage))
	r.GET("/api/goals/progress", api.GetGoalProgress(storage))
	return r, storage, goalsFile
}

func TestPostGoal_ValidAndInvalid(t *testing.T) {
	r, _, goalsFile := setupRouterAndStorage(t)
	ts := httptest.NewRecorder()
	// Valid
	body := `{"type":"duration","value":"7h"}`
	req, _ := http.NewRequest("POST", "/api/goals", ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 201, ts.Code)
	// Assert goals file exists and is not empty
	info, err := os.Stat(goalsFile)
	assert.NoError(t, err)
	assert.True(t, info.Size() > 0)
	// Invalid: missing value
	ts = httptest.NewRecorder()
	body = `{"type":"duration"}`
	req, _ = http.NewRequest("POST", "/api/goals", ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
	// Invalid: wrong type
	ts = httptest.NewRecorder()
	body = `{"type":"banana","value":"7h"}`
	req, _ = http.NewRequest("POST", "/api/goals", ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
}

func TestPostSleep_ValidAndInvalid(t *testing.T) {
	r, _, _ := setupRouterAndStorage(t)
	ts := httptest.NewRecorder()
	// Valid
	start := time.Now().Add(-8 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	body := `{"start_time":"` + start + `","end_time":"` + end + `","quality":7}`
	req, _ := http.NewRequest("POST", "/sleep", ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 201, ts.Code)
	// Invalid: quality out of range
	ts = httptest.NewRecorder()
	body = `{"start_time":"` + start + `","end_time":"` + end + `","quality":99}`
	req, _ = http.NewRequest("POST", "/sleep", ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
	// Invalid: missing start_time
	ts = httptest.NewRecorder()
	body = `{"end_time":"` + end + `","quality":7}`
	req, _ = http.NewRequest("POST", "/sleep", ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 400, ts.Code)
}

func TestGetGoalProgress_NoGoal(t *testing.T) {
	r, _, _ := setupRouterAndStorage(t)
	ts := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/goals/progress", nil)
	req.Header.Set("Authorization", "Bearer MOCK-TOKEN")
	r.ServeHTTP(ts, req)
	assert.Equal(t, 404, ts.Code)
}
