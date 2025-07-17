package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal"
)

func AuthMiddleware(storage *internal.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimPrefix(header, "Bearer ")
			token = strings.TrimSpace(token)
			user, err := storage.GetUserByToken(token)
			if err == nil {
				c.Set("user", user)
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	}
}
