package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal"
	"github.com/yourname/sleeptracker/internal/response"
)

func HandleError(c *gin.Context, logger internal.Logger, err error, status int, msg string) {
	requestID := c.GetString("request_id")
	logger.Errorf("[request_id=%s] %s: %v", requestID, msg, err)
	var resp response.APIResponse
	switch status {
	case 400:
		resp = response.BadRequest(msg + ": " + err.Error())
	case 404:
		resp = response.NotFound(msg + ": " + err.Error())
	case 500:
		resp = response.InternalError(msg + ": " + err.Error())
	default:
		resp = response.NewAppError(status, msg+": "+err.Error())
	}
	c.JSON(status, resp)
}

func HandleSuccess(c *gin.Context, logger internal.Logger, data interface{}, meta map[string]any) {
	requestID := c.GetString("request_id")
	logger.Infof("[request_id=%s] Success", requestID)
	c.JSON(200, response.Success(data, meta))
}
