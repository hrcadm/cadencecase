package storage

import (
	"context"

	"github.com/yourname/sleeptracker/internal"
)

type SleepLogRepository interface {
	SaveSleepLog(ctx context.Context, log *internal.SleepLog) error
	ListSleepLogs(ctx context.Context, userID string) ([]internal.SleepLog, error)
}

type GoalRepository interface {
	SetGoal(ctx context.Context, goal *internal.Goal) error
	GetGoal(ctx context.Context, userID string) (*internal.Goal, error)
}

type AuthProvider interface {
	ValidateTokenLocal(token string) (*internal.User, error)
	ValidateTokenRemote(ctx context.Context, token string) (*internal.User, error)
}
