package middleware

import (
	"awesomeProject/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogger logs all incoming HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Determine log level based on status code
		logLevel := logrus.InfoLevel
		if statusCode >= 500 {
			logLevel = logrus.ErrorLevel
		} else if statusCode >= 400 {
			logLevel = logrus.WarnLevel
		}

		// Log with structured fields
		fields := logrus.Fields{
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"status":       statusCode,
			"duration_ms":  duration.Milliseconds(),
			"ip":           c.ClientIP(),
			"user_agent":   c.Request.UserAgent(),
			"query":        c.Request.URL.RawQuery,
			"response_size": c.Writer.Size(),
		}

		// Add user info if authenticated
		if user, exists := c.Get("user"); exists {
			fields["user_id"] = user
		}

		// Log based on level
		utils.Log.WithFields(fields).Log(logLevel, "HTTP Request")
	}
}

// ErrorLogger logs errors with context
func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				utils.Log.WithFields(logrus.Fields{
					"error":  err.Error(),
					"type":   err.Type,
					"method": c.Request.Method,
					"path":   c.Request.URL.Path,
				}).Error("Request error occurred")
			}
		}
	}
}