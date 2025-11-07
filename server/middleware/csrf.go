package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

// CSRF token store with expiration
type csrfToken struct {
	Token     string
	CreatedAt time.Time
}

var (
	csrfTokens = make(map[string]csrfToken)
	csrfMutex  = &sync.RWMutex{}
)

// GenerateCSRFToken generates a new CSRF token
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	csrfMutex.Lock()
	csrfTokens[token] = csrfToken{
		Token:     token,
		CreatedAt: time.Now(),
	}
	csrfMutex.Unlock()

	// Clean up old tokens (older than 1 hour)
	go cleanupExpiredTokens()

	return token
}

// cleanupExpiredTokens removes tokens older than 1 hour
func cleanupExpiredTokens() {
	csrfMutex.Lock()
	defer csrfMutex.Unlock()

	now := time.Now()
	for token, data := range csrfTokens {
		if now.Sub(data.CreatedAt) > time.Hour {
			delete(csrfTokens, token)
		}
	}
}

// CSRFProtection middleware validates CSRF tokens for state-changing methods
func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods (GET, HEAD, OPTIONS)
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip for login and register endpoints (they don't have tokens yet)
		if c.Request.URL.Path == "/login" || c.Request.URL.Path == "/users" {
			c.Next()
			return
		}

		// Get token from header
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			// Try to get from form data
			token = c.PostForm("csrf_token")
		}

		if token == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			c.Abort()
			return
		}

		// Validate token
		csrfMutex.RLock()
		tokenData, exists := csrfTokens[token]
		csrfMutex.RUnlock()

		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
			c.Abort()
			return
		}

		// Check if token is expired (1 hour)
		if time.Since(tokenData.CreatedAt) > time.Hour {
			csrfMutex.Lock()
			delete(csrfTokens, token)
			csrfMutex.Unlock()

			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token expired"})
			c.Abort()
			return
		}

		// Token is valid, continue
		c.Next()

		// Optional: Remove token after use (single-use tokens)
		// Uncomment if you want single-use tokens:
		// csrfMutex.Lock()
		// delete(csrfTokens, token)
		// csrfMutex.Unlock()
	}
}

// GetCSRFTokenHandler endpoint handler to get a new CSRF token
func GetCSRFTokenHandler(c *gin.Context) {
	token := GenerateCSRFToken()
	c.JSON(http.StatusOK, gin.H{
		"csrf_token": token,
		"expires_in": 3600, // 1 hour in seconds
	})
}
