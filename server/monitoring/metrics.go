package monitoring

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	HttpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	HttpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	// Application metrics
	ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)

	TotalUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "total_users",
			Help: "Total number of registered users",
		},
	)

	TotalGames = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "total_games",
			Help: "Total number of games in catalog",
		},
	)

	AuthenticationAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authentication_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"status"}, // success or failure
	)

	// Error metrics
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "endpoint"},
	)
)

// InitMetrics initializes Prometheus metrics
func InitMetrics() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(HttpRequestSize)
	prometheus.MustRegister(HttpResponseSize)
	prometheus.MustRegister(ActiveConnections)
	prometheus.MustRegister(TotalUsers)
	prometheus.MustRegister(TotalGames)
	prometheus.MustRegister(AuthenticationAttempts)
	prometheus.MustRegister(ErrorsTotal)
}

// PrometheusMiddleware collects metrics for each request
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment active connections
		ActiveConnections.Inc()
		defer ActiveConnections.Dec()

		// Record request size
		HttpRequestSize.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(float64(c.Request.ContentLength))

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Record metrics
		status := c.Writer.Status()
		HttpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			string(rune(status)),
		).Inc()

		HttpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)

		HttpResponseSize.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(float64(c.Writer.Size()))

		// Record errors
		if status >= 400 {
			ErrorsTotal.WithLabelValues(
				"http_error",
				c.FullPath(),
			).Inc()
		}
	}
}

// PrometheusHandler returns Prometheus metrics handler
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}