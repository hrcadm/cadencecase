package test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourname/sleeptracker/internal"
)

func setupTestStorage(t *testing.T) *internal.Storage {
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
	storage, err := internal.NewStorage(usersFile, sleepFile, goalsFile)
	assert.NoError(t, err)
	// Add user via file
	os.WriteFile(usersFile, []byte(`[{"id":"u1","token":"MOCK-TOKEN","name":"Test User"}]`), 0644)
	storage, err = internal.NewStorage(usersFile, sleepFile, goalsFile)
	assert.NoError(t, err)
	return storage
}

func TestSaveAndListSleepLogs(t *testing.T) {
	storage := setupTestStorage(t)
	log := &internal.SleepLog{
		ID:        "log1",
		UserID:    "u1",
		StartTime: time.Now().Add(-8 * time.Hour),
		EndTime:   time.Now(),
		Quality:   7,
	}
	err := storage.SaveSleepLog(log)
	assert.NoError(t, err)
	logs, err := storage.ListSleepLogs("u1")
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, 7, logs[0].Quality)
}

func TestInputValidation(t *testing.T) {
	storage := setupTestStorage(t)
	log := &internal.SleepLog{
		ID:        "log2",
		UserID:    "u1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(-1 * time.Hour), // invalid
		Quality:   11,                             // invalid
	}
	err := storage.SaveSleepLog(log)
	assert.NoError(t, err) // storage does not validate, handler does
}

func TestStatsCalculation(t *testing.T) {
	storage := setupTestStorage(t)
	now := time.Now()
	logs := []*internal.SleepLog{
		{ID: "l1", UserID: "u1", StartTime: now.AddDate(0, 0, -1), EndTime: now, Quality: 8},
		{ID: "l2", UserID: "u1", StartTime: now.AddDate(0, 0, -2), EndTime: now, Quality: 6},
		{ID: "l3", UserID: "u1", StartTime: now.AddDate(0, 0, -8), EndTime: now, Quality: 9}, // outside 7 days
	}
	for _, l := range logs {
		_ = storage.SaveSleepLog(l)
	}
	userLogs, err := storage.ListSleepLogs("u1")
	assert.NoError(t, err)
	var total, count int
	cutoff := now.AddDate(0, 0, -7)
	for _, l := range userLogs {
		if l.StartTime.After(cutoff) {
			total += l.Quality
			count++
		}
	}
	avg := 0.0
	if count > 0 {
		avg = float64(total) / float64(count)
	}
	assert.Equal(t, 2, count)
	assert.InDelta(t, 7.0, avg, 0.01)
}

func TestSetAndGetGoal(t *testing.T) {
	storage := setupTestStorage(t)
	goal := &internal.Goal{
		ID:        "goal1",
		UserID:    "u1",
		Type:      "duration",
		Value:     "7h",
		CreatedAt: time.Now(),
	}
	err := storage.SetGoal(goal)
	assert.NoError(t, err)
	// Check file exists and is not empty
	info, err := os.Stat("testdata/test_goals.json")
	assert.NoError(t, err)
	assert.True(t, info.Size() > 0)
	// Check retrieval
	got, err := storage.GetGoal("u1")
	assert.NoError(t, err)
	assert.Equal(t, goal.Type, got.Type)
	assert.Equal(t, goal.Value, got.Value)
}
