package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

type Storage struct {
	users          map[string]*User
	sleepLogs      map[string]*SleepLog   // id -> SleepLog
	userSleepIndex map[string][]*SleepLog // userID -> slice of SleepLogs (sorted descending)
	goals          map[string]*Goal       // userID -> Goal
	mu             sync.RWMutex
	usersFile      string
	sleepFile      string
	goalsFile      string
	saveLogsChan   chan struct{}
	saveGoalsChan  chan struct{}
	shutdownChan   chan struct{}
	saveLogsDelay  time.Duration
	saveGoalsDelay time.Duration
}

// NewStorage initializes Storage, loads data, and starts save workers.
func NewStorage(usersFile, sleepFile, goalsFile string) (*Storage, error) {
	s := &Storage{
		users:          make(map[string]*User),
		sleepLogs:      make(map[string]*SleepLog),
		userSleepIndex: make(map[string][]*SleepLog),
		goals:          make(map[string]*Goal),
		usersFile:      usersFile,
		sleepFile:      sleepFile,
		goalsFile:      goalsFile,
		saveLogsChan:   make(chan struct{}, 1),
		saveGoalsChan:  make(chan struct{}, 1),
		shutdownChan:   make(chan struct{}),
		saveLogsDelay:  500 * time.Millisecond,
		saveGoalsDelay: 500 * time.Millisecond,
	}

	if err := s.loadUsers(); err != nil {
		return nil, fmt.Errorf("storage: failed to load users: %w", err)
	}
	if err := s.loadSleepLogs(); err != nil {
		return nil, fmt.Errorf("storage: failed to load sleep logs: %w", err)
	}
	if err := s.loadGoals(); err != nil {
		return nil, fmt.Errorf("storage: failed to load goals: %w", err)
	}

	go s.saveLogsWorker()
	go s.saveGoalsWorker()

	return s, nil
}

func (s *Storage) loadUsers() error {
	file, err := os.Open(s.usersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var users []*User
	if err := json.NewDecoder(file).Decode(&users); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range users {
		s.users[u.Token] = u
	}
	return nil
}

func (s *Storage) loadSleepLogs() error {
	file, err := os.Open(s.sleepFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var logs []*SleepLog
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

func (s *Storage) loadGoals() error {
	file, err := os.Open(s.goalsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var goals []*Goal
	if err := json.NewDecoder(file).Decode(&goals); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range goals {
		s.goals[g.UserID] = g
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

func (s *Storage) saveSleepLogs() error {
	s.mu.RLock()
	logs := make([]*SleepLog, 0, len(s.sleepLogs))
	for _, l := range s.sleepLogs {
		logs = append(logs, l)
	}
	s.mu.RUnlock()

	return atomicWriteFileJSON(s.sleepFile, logs)
}

func (s *Storage) saveGoals() error {
	s.mu.RLock()
	goals := make([]*Goal, 0, len(s.goals))
	for _, g := range s.goals {
		goals = append(goals, g)
	}
	s.mu.RUnlock()

	return atomicWriteFileJSON(s.goalsFile, goals)
}

// saveLogsWorker batches save operations to avoid frequent disk writes
func (s *Storage) saveLogsWorker() {
	timer := time.NewTimer(s.saveLogsDelay)
	defer timer.Stop()

	for {
		select {
		case <-s.saveLogsChan:
			timer.Reset(s.saveLogsDelay)
		case <-timer.C:
			if err := s.saveSleepLogs(); err != nil {
				// Log the error somewhere or handle accordingly
				fmt.Printf("storage: error saving sleep logs: %v\n", err)
			}
		case <-s.shutdownChan:
			return
		}
	}
}

// saveGoalsWorker batches goal saves
func (s *Storage) saveGoalsWorker() {
	timer := time.NewTimer(s.saveGoalsDelay)
	defer timer.Stop()

	for {
		select {
		case <-s.saveGoalsChan:
			timer.Reset(s.saveGoalsDelay)
		case <-timer.C:
			if err := s.saveGoals(); err != nil {
				fmt.Printf("storage: error saving goals: %v\n", err)
			}
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *Storage) SetGoal(goal *Goal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.goals[goal.UserID] = goal

	// Signal the worker to save goals
	select {
	case s.saveGoalsChan <- struct{}{}:
	default:
	}

	return nil
}

func (s *Storage) GetGoal(userID string) (*Goal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.goals[userID]
	if !ok {
		return nil, fmt.Errorf("storage: goal not found")
	}
	return g, nil
}

func (s *Storage) GetUserByToken(token string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[token]
	if !ok {
		return nil, fmt.Errorf("storage: user not found")
	}
	return u, nil
}

func (s *Storage) SaveSleepLog(log *SleepLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update main map
	s.sleepLogs[log.ID] = log

	// Update user index - insert maintaining descending order
	logs := s.userSleepIndex[log.UserID]
	inserted := false
	for i, existing := range logs {
		if existing.StartTime.Before(log.StartTime) {
			// Insert here
			logs = append(logs[:i], append([]*SleepLog{log}, logs[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		logs = append(logs, log)
	}
	s.userSleepIndex[log.UserID] = logs

	// Signal the save worker (non-blocking)
	select {
	case s.saveLogsChan <- struct{}{}:
	default:
	}

	return nil
}

func (s *Storage) ListSleepLogs(userID string) ([]SleepLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logsPtr, ok := s.userSleepIndex[userID]
	if !ok {
		return []SleepLog{}, nil
	}

	logs := make([]SleepLog, len(logsPtr))
	for i, l := range logsPtr {
		logs[i] = *l
	}

	return logs, nil
}

// Close storage and stop background workers gracefully
func (s *Storage) Close() error {
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
