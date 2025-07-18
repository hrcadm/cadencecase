package api

import (
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/storage"
)

type App interface {
	Logger() internal.Logger
	SleepRepo() storage.SleepLogRepository
	GoalRepo() storage.GoalRepository
}
