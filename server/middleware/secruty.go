package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection (legacy browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		// Allows resources from same origin and localhost for development
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: http://localhost:8080 https://localhost:8080; "+
				"font-src 'self'; "+
				"connect-src 'self' http://localhost:8080 https://localhost:8080; "+
				"frame-ancestors 'none'")

		// Referrer Policy - controls how much referrer information is sent
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy - controls which browser features can be used
		c.Header("Permissions-Policy",
			"geolocation=(), "+
				"microphone=(), "+
				"camera=(), "+
				"payment=(), "+
				"usb=(), "+
				"magnetometer=(), "+
				"gyroscope=(), "+
				"accelerometer=()")

		// Strict Transport Security (HSTS) - only if using HTTPS
		// Tells browsers to always use HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Expect-CT - Certificate Transparency
		if c.Request.TLS != nil {
			c.Header("Expect-CT", "max-age=86400, enforce")
		}

		c.Next()
	}
}

// RateLimitInfo adds rate limit information to response headers (informational)
func RateLimitInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-RateLimit-Limit", "1000")
		c.Header("X-RateLimit-Window", "1h")
		c.Next()
	}
}

// RemovePoweredBy removes or modifies the Server header
func RemovePoweredBy() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Remove default server information
		c.Header("Server", "")
		c.Next()
	}
}