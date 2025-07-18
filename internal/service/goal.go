package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/storage"
)

type GoalRequest struct {
	Type  string `validate:"required,oneof=duration consistency quality"`
	Value string `validate:"required"`
}

type GoalProgress struct {
	Goal      *internal.Goal
	Progress  []map[string]interface{}
	MetDays   int
	TotalDays int
}

func ValidateGoalRequest(req *GoalRequest) error {
	if err := validate.Struct(req); err != nil {
		return err
	}
	return nil
}

func CreateGoal(ctx context.Context, goalRepo storage.GoalRepository, user *internal.User, req *GoalRequest) (*internal.Goal, error) {
	goal := &internal.Goal{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Type:      req.Type,
		Value:     req.Value,
		CreatedAt: time.Now(),
	}
	if err := goalRepo.SetGoal(ctx, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func CalculateGoalProgress(goal *internal.Goal, logs []internal.SleepLog) GoalProgress {
	cutoff := time.Now().AddDate(0, 0, -7)
	days := []map[string]interface{}{}
	metCount := 0

	for _, l := range logs {
		if l.StartTime.Before(cutoff) {
			break
		}

		met := false
		switch goal.Type {
		case "duration":
			var durGoal float64
			fmt.Sscanf(goal.Value, "%fh", &durGoal)
			sleepDur := l.EndTime.Sub(l.StartTime).Hours()
			met = sleepDur >= durGoal
		case "consistency":
			var hour int
			fmt.Sscanf(goal.Value, "before %d", &hour)
			if hour == 0 {
				hour = 23
			}
			met = l.StartTime.Hour() < hour
		case "quality":
			var qualGoal int
			fmt.Sscanf(goal.Value, "> %d", &qualGoal)
			met = l.Quality > qualGoal
		}

		if met {
			metCount++
		}

		days = append(days, map[string]interface{}{
			"date": l.StartTime.Format("2006-01-02"),
			"met":  met,
		})
	}

	return GoalProgress{
		Goal:      goal,
		Progress:  days,
		MetDays:   metCount,
		TotalDays: len(days),
	}
}
