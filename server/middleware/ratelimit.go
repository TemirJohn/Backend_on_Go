package middleware

//
//import (
//	"awesomeProject/cache"
//	"awesomeProject/models"
//	"fmt"
//	"net/http"
//	"strconv"
//	"time"
//
//	"github.com/gin-gonic/gin"
//)
//
//// RateLimitMiddleware implements per-user rate limiting using Redis
//func RateLimitMiddleware(maxRequests int, window time.Duration) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		// Skip rate limiting if Redis is not available
//		if !cache.IsRedisAvailable() {
//			c.Next()
//			return
//		}
//
//		// Get user from context (if authenticated)
//		var userID uint
//		if user, exists := c.Get("user"); exists {
//			if u, ok := user.(models.User); ok {
//				userID = u.ID
//			}
//		}
//
//		// For unauthenticated users, use IP address
//		if userID == 0 {
//			// Use IP-based rate limiting
//			ip := c.ClientIP()
//			allowed, remaining, err := cache.CheckRateLimit(
//				hashIP(ip),
//				maxRequests,
//				window,
//			)
//
//			if err != nil {
//				c.JSON(http.StatusTooManyRequests, gin.H{
//					"error": "Rate limit exceeded",
//				})
//				c.Abort()
//				return
//			}
//
//			// Set rate limit headers
//			c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequests))
//			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
//			c.Header("X-RateLimit-Window", window.String())
//
//			if !allowed {
//				c.JSON(http.StatusTooManyRequests, gin.H{
//					"error":   "Rate limit exceeded",
//					"message": fmt.Sprintf("Too many requests. Retry after %v", window),
//				})
//				c.Abort()
//				return
//			}
//		} else {
//			// User-based rate limiting
//			allowed, remaining, err := cache.CheckRateLimit(userID, maxRequests, window)
//
//			if err != nil {
//				c.JSON(http.StatusTooManyRequests, gin.H{
//					"error": "Rate limit exceeded",
//				})
//				c.Abort()
//				return
//			}
//
//			// Set rate limit headers
//			c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequests))
//			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
//			c.Header("X-RateLimit-Window", window.String())
//
//			if !allowed {
//				c.JSON(http.StatusTooManyRequests, gin.H{
//					"error":   "Rate limit exceeded",
//					"message": fmt.Sprintf("Too many requests. Retry after %v", window),
//				})
//				c.Abort()
//				return
//			}
//		}
//
//		c.Next()
//	}
//}
//
//// hashIP converts IP to a simple numeric ID for caching
//func hashIP(ip string) uint {
//	hash := uint(0)
//	for _, c := range ip {
//		hash = hash*31 + uint(c)
//	}
//	return hash
//}
//
//// RateLimitPerEndpoint implements endpoint-specific rate limiting
//func RateLimitPerEndpoint(limits map[string]struct {
//	MaxRequests int
//	Window      time.Duration
//}) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		endpoint := c.FullPath()
//
//		// Get rate limit config for this endpoint
//		config, exists := limits[endpoint]
//		if !exists {
//			// No specific limit for this endpoint
//			c.Next()
//			return
//		}
//
//		// Get user ID
//		var userID uint
//		if user, exists := c.Get("user"); exists {
//			if u, ok := user.(models.User); ok {
//				userID = u.ID
//			}
//		}
//
//		if userID == 0 {
//			userID = hashIP(c.ClientIP())
//		}
//
//		// Create endpoint-specific cache key
//
//		allowed, remaining, err := cache.CheckRateLimit(
//			userID,
//			config.MaxRequests,
//			config.Window,
//		)
//
//		if err != nil || !allowed {
//			c.JSON(http.StatusTooManyRequests, gin.H{
//				"error":   "Rate limit exceeded for this endpoint",
//				"message": fmt.Sprintf("Retry after %v", config.Window),
//			})
//			c.Abort()
//			return
//		}
//
//		c.Header("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
//		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
//
//		c.Next()
//	}
//}
