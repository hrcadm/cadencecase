package api

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourname/sleeptracker/internal"
)

// --- Request Structs ---
type SleepLogRequest struct {
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Quality       int       `json:"quality"`
	Reason        string    `json:"reason,omitempty"`
	Interruptions []string  `json:"interruptions,omitempty"`
}

type GoalRequest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// --- Handlers ---
func PostSleep(storage *internal.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)

		var body SleepLogRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error(), "code": 400})
			return
		}

		if body.EndTime.Before(body.StartTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "'end_time' must be after 'start_time'", "code": 400})
			return
		}

		if body.Quality < 1 || body.Quality > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "'quality' must be an integer between 1 and 10", "code": 400})
			return
		}

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

		if err := storage.SaveSleepLog(log); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save log: " + err.Error(), "code": 500})
			return
		}

		c.JSON(http.StatusCreated, log)
	}
}

func GetSleep(storage *internal.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)

		logs, err := storage.ListSleepLogs(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, internal.NewAppError(500, "Failed to fetch logs: "+err.Error()))
			return
		}

		sort.Slice(logs, func(i, j int) bool {
			return logs[i].StartTime.After(logs[j].StartTime)
		})

		c.JSON(http.StatusOK, logs)
	}
}

func GetSleepStats(storage *internal.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)
		logs, err := storage.ListSleepLogs(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, internal.NewAppError(500, "Failed to fetch logs: "+err.Error()))
			return
		}

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

		c.JSON(http.StatusOK, gin.H{
			"average_quality": avg,
			"trend":           trend,
		})
	}
}

func GetSleepRecommendations() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"recommendation": "Try to maintain a consistent sleep schedule.",
			"reason":         "Regular sleep improves quality.",
			"action":         "Go to bed and wake up at the same time every day.",
			"source":         "MockGPT",
		})
	}
}

func PostGoal(storage *internal.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)

		var req GoalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: type and value required", "code": 400})
			return
		}

		supported := map[string]bool{"duration": true, "consistency": true, "quality": true}
		if !supported[req.Type] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid goal type", "code": 400})
			return
		}

		if req.Value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Goal value required", "code": 400})
			return
		}

		goal := &internal.Goal{
			ID:        uuid.NewString(),
			UserID:    user.ID,
			Type:      req.Type,
			Value:     req.Value,
			CreatedAt: time.Now(),
		}

		if err := storage.SetGoal(goal); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save goal: " + err.Error(), "code": 500})
			return
		}

		c.JSON(http.StatusCreated, goal)
	}
}

func GetGoalProgress(storage *internal.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)
		goal, err := storage.GetGoal(user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No goal set for user", "code": 404})
			return
		}

		logs, err := storage.ListSleepLogs(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs: " + err.Error(), "code": 500})
			return
		}

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

		c.JSON(http.StatusOK, gin.H{
			"goal":       goal,
			"progress":   days,
			"met_days":   metCount,
			"total_days": len(days),
		})
	}
}
