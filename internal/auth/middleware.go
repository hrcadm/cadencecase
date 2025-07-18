package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourname/sleeptracker/internal/config"
)

func AuthMiddleware(provider Provider, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimPrefix(header, "Bearer ")
			token = strings.TrimSpace(token)
			var user interface{}
			var err error
			if cfg.Env == "development" {
				user, err = provider.ValidateTokenLocal(token)
			} else {
				user, err = provider.ValidateTokenRemote(c.Request.Context(), token)
			}
			if err == nil {
				c.Set("user", user)
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	}
}
