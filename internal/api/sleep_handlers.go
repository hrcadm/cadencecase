package api

import (
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/service"
)

func PostSleep(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)

		var body service.SleepLogRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			HandleError(c, app.Logger(), err, 400, "Invalid JSON")
			return
		}
		app.Logger().Infof("Parsed SleepLogRequest: %+v", body)

		if err := service.ValidateSleepLogRequest(&body); err != nil {
			HandleError(c, app.Logger(), err, 400, "Validation failed")
			return
		}

		log, err := service.CreateSleepLog(c.Request.Context(), app.SleepRepo(), user, &body)
		if err != nil {
			HandleError(c, app.Logger(), err, 500, "Failed to save log")
			return
		}

		HandleSuccess(c, app.Logger(), log, nil)
	}
}

func GetSleep(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)

		logs, err := app.SleepRepo().ListSleepLogs(c.Request.Context(), user.ID)
		if err != nil {
			HandleError(c, app.Logger(), err, 500, "Failed to fetch logs")
			return
		}

		sort.Slice(logs, func(i, j int) bool {
			return logs[i].StartTime.After(logs[j].StartTime)
		})

		HandleSuccess(c, app.Logger(), logs, nil)
	}
}

func GetSleepStats(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)
		logs, err := app.SleepRepo().ListSleepLogs(c.Request.Context(), user.ID)
		if err != nil {
			HandleError(c, app.Logger(), err, 500, "Failed to fetch logs for stats")
			return
		}

		avg, trend := service.CalculateSleepStats(logs)
		meta := map[string]any{"average_quality": avg, "trend": trend}
		HandleSuccess(c, app.Logger(), nil, meta)
	}
}

func GetSleepRecommendations(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		meta := map[string]any{
			"recommendation": "Try to maintain a consistent sleep schedule.",
			"reason":         "Regular sleep improves quality.",
			"action":         "Go to bed and wake up at the same time every day.",
			"source":         "MockGPT",
		}
		HandleSuccess(c, app.Logger(), nil, meta)
	}
}
