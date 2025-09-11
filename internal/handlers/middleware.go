package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// AuthMiddleware creates JWT authentication middleware
func AuthMiddleware(userService user.UserService, logger *Logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := userService.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			logger.Debugf("token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("claims", claims)

		c.Next()
	}
}

// AdminMiddleware checks if user has admin role
// Note: This is a placeholder - you'd need to implement role-based access control
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, we'll skip admin checks
		// In a real implementation, you'd check user roles from database or JWT claims

		// Example implementation:
		// userID := c.GetString("userID")
		// if !isUserAdmin(userID) {
		//     c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		//     c.Abort()
		//     return
		// }

		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestLoggerMiddleware logs incoming requests
func RequestLoggerMiddleware(logger *Logger.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Infof("[%s] %s %s %d %s %s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
		return ""
	})
}

// ErrorHandlerMiddleware handles panics and errors
func ErrorHandlerMiddleware(logger *Logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Errorf("Panic recovered: %v", recovered)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	})
}
