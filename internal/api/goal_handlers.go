package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/service"
)

func PostGoal(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)

		var req service.GoalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleError(c, app.Logger(), err, 400, "Invalid request: type and value required")
			return
		}

		if err := service.ValidateGoalRequest(&req); err != nil {
			HandleError(c, app.Logger(), err, 400, "Goal validation failed")
			return
		}

		goal, err := service.CreateGoal(c.Request.Context(), app.GoalRepo(), user, &req)
		if err != nil {
			HandleError(c, app.Logger(), err, 500, "Failed to save goal")
			return
		}

		HandleSuccess(c, app.Logger(), goal, nil)
	}
}

func GetGoalProgress(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*internal.User)
		goal, err := app.GoalRepo().GetGoal(c.Request.Context(), user.ID)
		if err != nil {
			HandleError(c, app.Logger(), err, 404, "No goal set for user")
			return
		}

		logs, err := app.SleepRepo().ListSleepLogs(c.Request.Context(), user.ID)
		if err != nil {
			HandleError(c, app.Logger(), err, 500, "Failed to fetch logs for goal progress")
			return
		}

		progress := service.CalculateGoalProgress(goal, logs)
		HandleSuccess(c, app.Logger(), progress, nil)
	}
}
