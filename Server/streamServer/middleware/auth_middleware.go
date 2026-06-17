package middleware

import (
	"net/http"
	"os"
	"stream-server/utils"

	"github.com/gin-gonic/gin"
)

var TOKEN_SECRET_KEY = os.Getenv("TOKEN_SECRET_KEY")

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// get token from the header
		token, err := utils.GetAccessToken(ctx)

		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			ctx.Abort()
			return
		}

		if token == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			ctx.Abort()
			return
		}

		// validate token
		user, err := utils.ValidateToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			ctx.Abort()
			return
		}

		ctx.Set("user_id", user.UserID)
		ctx.Set("role", user.Role)
		ctx.Next()
	}
}
