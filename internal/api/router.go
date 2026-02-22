package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NewRouter initializes and returns a configured *gin.Engine router.
// It sets up global middleware and registers application routes.
func NewRouter() *gin.Engine {
	// Use gin.New() instead of Default() to explicitly control middleware
	r := gin.New()

	// Global Middleware
	r.Use(gin.Recovery())
	// In production, we'd replace this with a structured logger (slog/zap)
	r.Use(gin.Logger())

	// API Versioning Group
	v1 := r.Group("/api/v1")
	{
		// Health & Readiness Endpoints (Critical for Kubernetes)
		v1.GET("/health", healthCheckHandler)
		v1.GET("/readiness", readinessHandler)

		// Temporary ping endpoint (can be removed later)
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	return r
}

// healthCheckHandler responds indicating the process is alive.
// Kubernetes uses this to know if the pod is running.
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// readinessHandler responds indicating if the app is ready to take traffic.
// Kubernetes uses this to know if it should route traffic to this pod.
// E.g., This might fail if the database connection drops.
func readinessHandler(c *gin.Context) {
	// TODO: Add database ping/connection check here.
	// For now, we return UP.
	c.JSON(http.StatusOK, gin.H{
		"status":  "READY",
		"message": "Service is ready to accept traffic",
	})
}
