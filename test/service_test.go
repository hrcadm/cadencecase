package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/auth"
	"github.com/yourname/sleeptracker/internal/storage"
	"go.uber.org/zap"
)

func setupFileStorage(t *testing.T) storage.SleepLogRepository {
	testDir := "testdata"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		_ = os.MkdirAll(testDir, 0755)
	}
	sleepFile := testDir + "/test_sleep_logs.json"
	goalsFile := testDir + "/test_goals.json"
	os.Remove(sleepFile)
	os.Remove(goalsFile)
	repo, _, err := storage.NewFileRepositories(sleepFile, goalsFile, internal.NewZapLogger(zap.NewNop().Sugar()))
	assert.NoError(t, err)
	return repo
}

func TestFileStorageCRUD(t *testing.T) {
	repo := setupFileStorage(t)
	ctx := context.Background()
	log := &internal.SleepLog{
		ID:        "log1",
		UserID:    "u1",
		StartTime: time.Now().Add(-8 * time.Hour),
		EndTime:   time.Now(),
		Quality:   7,
	}
	err := repo.SaveSleepLog(ctx, log)
	assert.NoError(t, err)
	logs, err := repo.ListSleepLogs(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, 7, logs[0].Quality)
}

func TestPostgresStorageReady(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set, skipping Postgres test")
	}
	logger := internal.NewZapLogger(zap.NewNop().Sugar())
	repo, _, err := storage.NewPostgresRepositories(dsn, logger)
	assert.NoError(t, err)
	ctx := context.Background()
	// Just check that ListSleepLogs does not error (even if empty)
	_, err = repo.ListSleepLogs(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestLocalAuthProvider(t *testing.T) {
	logger := internal.NewZapLogger(zap.NewNop().Sugar())
	provider := auth.NewLocalAuthProvider("MOCK-TOKEN", logger)
	user, err := provider.ValidateTokenLocal("MOCK-TOKEN")
	assert.NoError(t, err)
	assert.Equal(t, "u1", user.ID)
	_, err = provider.ValidateTokenLocal("WRONG-TOKEN")
	assert.Error(t, err)
}

func TestRemoteAuthProviderReady(t *testing.T) {
	// Mock remote auth service
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Token string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Token == "MOCK-TOKEN" {
			json.NewEncoder(w).Encode(&internal.User{ID: "u2", Token: req.Token, Name: "Remote User"})
		} else {
			w.WriteHeader(401)
		}
	}))
	defer ts.Close()
	logger := internal.NewZapLogger(zap.NewNop().Sugar())
	provider := auth.NewRemoteAuthProvider(ts.URL, logger)
	ctx := context.Background()
	user, err := provider.ValidateTokenRemote(ctx, "MOCK-TOKEN")
	assert.NoError(t, err)
	assert.Equal(t, "u2", user.ID)
	_, err = provider.ValidateTokenRemote(ctx, "WRONG-TOKEN")
	assert.Error(t, err)
}
