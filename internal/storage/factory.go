package storage

import "github.com/yourname/sleeptracker/internal"

func NewFileRepositories(sleepFile, goalsFile string, logger internal.Logger) (SleepLogRepository, GoalRepository, error) {
	storage, err := NewFileStorage(sleepFile, goalsFile, logger)
	if err != nil {
		return nil, nil, err
	}
	return storage, storage, nil
}

func NewPostgresRepositories(dsn string, logger internal.Logger) (SleepLogRepository, GoalRepository, error) {
	storage, err := NewPostgresStorage(dsn, logger)
	if err != nil {
		return nil, nil, err
	}
	return storage, storage, nil
}
