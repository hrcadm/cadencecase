package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/yourname/sleeptracker/internal"
)

type FileStorage struct {
	sleepLogs      map[string]*internal.SleepLog        // id -> SleepLog
	userSleepIndex map[string][]*internal.SleepLog      // userID -> slice of SleepLogs (sorted descending)
	goals          map[string]map[string]*internal.Goal // userID -> type -> Goal
	mu             sync.RWMutex
	sleepFile      string
	goalsFile      string
	saveLogsChan   chan struct{}
	saveGoalsChan  chan struct{}
	shutdownChan   chan struct{}
	saveLogsDelay  time.Duration
	saveGoalsDelay time.Duration
	logger         internal.Logger
}

func NewFileStorage(sleepFile, goalsFile string, logger internal.Logger) (*FileStorage, error) {
	s := &FileStorage{
		sleepLogs:      make(map[string]*internal.SleepLog),
		userSleepIndex: make(map[string][]*internal.SleepLog),
		goals:          make(map[string]map[string]*internal.Goal),
		sleepFile:      sleepFile,
		goalsFile:      goalsFile,
		saveLogsChan:   make(chan struct{}, 1),
		saveGoalsChan:  make(chan struct{}, 1),
		shutdownChan:   make(chan struct{}),
		saveLogsDelay:  500 * time.Millisecond,
		saveGoalsDelay: 500 * time.Millisecond,
		logger:         logger,
	}

	if err := s.loadSleepLogs(); err != nil {
		logger.Errorf("storage: failed to load sleep logs: %v", err)
		return nil, err
	}
	if err := s.loadGoals(); err != nil {
		logger.Errorf("storage: failed to load goals: %v", err)
		return nil, err
	}

	go s.saveLogsWorker()
	go s.saveGoalsWorker()

	return s, nil
}

func (s *FileStorage) loadSleepLogs() error {
	file, err := os.Open(s.sleepFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var logs []*internal.SleepLog
	if err := json.NewDecoder(file).Decode(&logs); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range logs {
		s.sleepLogs[l.ID] = l
		s.userSleepIndex[l.UserID] = append(s.userSleepIndex[l.UserID], l)
	}

	// Sort each user's logs descending by StartTime
	for userID := range s.userSleepIndex {
		sort.Slice(s.userSleepIndex[userID], func(i, j int) bool {
			return s.userSleepIndex[userID][i].StartTime.After(s.userSleepIndex[userID][j].StartTime)
		})
	}

	return nil
}

func (s *FileStorage) loadGoals() error {
	file, err := os.Open(s.goalsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var goals []*internal.Goal
	if err := json.NewDecoder(file).Decode(&goals); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.goals = make(map[string]map[string]*internal.Goal)
	for _, g := range goals {
		if s.goals[g.UserID] == nil {
			s.goals[g.UserID] = make(map[string]*internal.Goal)
		}
		s.goals[g.UserID][g.Type] = g
	}

	return nil
}

func atomicWriteFileJSON(filePath string, data interface{}) error {
	tempFile := filePath + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		f.Close()
		os.Remove(tempFile)
		return err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tempFile)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tempFile)
		return err
	}

	return os.Rename(tempFile, filePath)
}

func (s *FileStorage) saveSleepLogs() error {
	s.mu.RLock()
	logs := make([]*internal.SleepLog, 0, len(s.sleepLogs))
	for _, l := range s.sleepLogs {
		logs = append(logs, l)
	}
	s.mu.RUnlock()

	return atomicWriteFileJSON(s.sleepFile, logs)
}

func (s *FileStorage) saveGoals() error {
	s.mu.RLock()
	var goals []*internal.Goal
	for _, typeMap := range s.goals {
		for _, g := range typeMap {
			goals = append(goals, g)
		}
	}
	s.mu.RUnlock()
	if goals == nil {
		goals = make([]*internal.Goal, 0)
	}
	return atomicWriteFileJSON(s.goalsFile, goals)
}

func (s *FileStorage) saveLogsWorker() {
	timer := time.NewTimer(s.saveLogsDelay)
	defer timer.Stop()

	for {
		select {
		case <-s.saveLogsChan:
			timer.Reset(s.saveLogsDelay)
		case <-timer.C:
			if err := s.saveSleepLogs(); err != nil {
				s.logger.Errorf("storage: error saving sleep logs: %v", err)
			}
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *FileStorage) saveGoalsWorker() {
	timer := time.NewTimer(s.saveGoalsDelay)
	defer timer.Stop()

	for {
		select {
		case <-s.saveGoalsChan:
			timer.Reset(s.saveGoalsDelay)
		case <-timer.C:
			if err := s.saveGoals(); err != nil {
				s.logger.Errorf("storage: error saving goals: %v", err)
			}
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *FileStorage) Close() error {
	close(s.shutdownChan)

	// Save pending data synchronously on shutdown
	if err := s.saveSleepLogs(); err != nil {
		return err
	}
	if err := s.saveGoals(); err != nil {
		return err
	}
	return nil
}

// --- SleepLogRepository ---
func (s *FileStorage) SaveSleepLog(ctx context.Context, log *internal.SleepLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sleepLogs[log.ID] = log
	logs := s.userSleepIndex[log.UserID]
	inserted := false
	for i, existing := range logs {
		if existing.StartTime.Before(log.StartTime) {
			logs = append(logs[:i], append([]*internal.SleepLog{log}, logs[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		logs = append(logs, log)
	}
	s.userSleepIndex[log.UserID] = logs
	select {
	case s.saveLogsChan <- struct{}{}:
	default:
	}
	return nil
}

func (s *FileStorage) ListSleepLogs(ctx context.Context, userID string) ([]internal.SleepLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	logsPtr, ok := s.userSleepIndex[userID]
	if !ok {
		return []internal.SleepLog{}, nil
	}
	logs := make([]internal.SleepLog, len(logsPtr))
	for i, l := range logsPtr {
		logs[i] = *l
	}
	return logs, nil
}

// --- GoalRepository ---
func (s *FileStorage) SetGoal(ctx context.Context, goal *internal.Goal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.goals[goal.UserID] == nil {
		s.goals[goal.UserID] = make(map[string]*internal.Goal)
	}
	s.goals[goal.UserID][goal.Type] = goal
	select {
	case s.saveGoalsChan <- struct{}{}:
	default:
	}
	return nil
}

func (s *FileStorage) GetGoal(ctx context.Context, userID string) (*internal.Goal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	typeMap, ok := s.goals[userID]
	if !ok || len(typeMap) == 0 {
		return nil, errors.New("storage: goal not found")
	}
	// Return the most recently created goal (by CreatedAt) among all types
	var latest *internal.Goal
	for _, g := range typeMap {
		if latest == nil || g.CreatedAt.After(latest.CreatedAt) {
			latest = g
		}
	}
	return latest, nil
}

// --- Compile-time assertions ---
var _ SleepLogRepository = (*FileStorage)(nil)
var _ GoalRepository = (*FileStorage)(nil)
