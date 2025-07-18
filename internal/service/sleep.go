package service

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/storage"
)

var validate = validator.New()

type SleepLogRequest struct {
	StartTime     time.Time `json:"start_time" validate:"required"`
	EndTime       time.Time `json:"end_time" validate:"required,gtfield=StartTime"`
	Quality       int       `json:"quality" validate:"required,gte=1,lte=10"`
	Reason        string    `json:"reason,omitempty" validate:"omitempty"`
	Interruptions []string  `json:"interruptions,omitempty" validate:"dive,required"`
}

func ValidateSleepLogRequest(body *SleepLogRequest) error {
	return validate.Struct(body)
}

func CreateSleepLog(ctx context.Context, sleepRepo storage.SleepLogRepository, user *internal.User, body *SleepLogRequest) (*internal.SleepLog, error) {
	log := &internal.SleepLog{
		ID:            uuid.NewString(),
		UserID:        user.ID,
		StartTime:     body.StartTime,
		EndTime:       body.EndTime,
		Quality:       body.Quality,
		Reason:        body.Reason,
		Interruptions: body.Interruptions,
		CreatedAt:     time.Now(),
	}
	if err := sleepRepo.SaveSleepLog(ctx, log); err != nil {
		return nil, err
	}
	return log, nil
}

func CalculateSleepStats(logs []internal.SleepLog) (float64, []int) {
	cutoff := time.Now().AddDate(0, 0, -7)
	totalQuality := 0
	count := 0
	trend := []int{}

	for _, l := range logs {
		if l.StartTime.After(cutoff) {
			totalQuality += l.Quality
			count++
			trend = append(trend, l.Quality)
		}
	}

	avg := 0.0
	if count > 0 {
		avg = float64(totalQuality) / float64(count)
	}

	return avg, trend
}
