package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db *pgxpool.Pool
}

func NewAPI(db *pgxpool.Pool) *API {
	return &API{db: db}
}

// NewRouter initializes and returns a configured *gin.Engine router.
// It sets up global middleware and registers application routes.
func (a *API) NewRouter() *gin.Engine {
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
		v1.GET("/health", a.healthCheckHandler)
		v1.GET("/readiness", a.readinessHandler)

		// Temporary ping endpoint (can be removed later)
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Workflow Trigger Endpoint
		v1.POST("/workflows/:id/trigger", a.workflowTriggerHandler)
	}

	return r
}

// healthCheckHandler responds indicating the process is alive.
// Kubernetes uses this to know if the pod is running.
func (a *API) healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// readinessHandler responds indicating if the app is ready to take traffic.
// E.g., This might fail if the database connection drops.
func (a *API) readinessHandler(c *gin.Context) {
	if err := a.db.Ping(c); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "NOT_READY",
			"message": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "READY",
		"message": "Service is ready to accept traffic",
	})
}

// workflowTriggerHandler initiates the execution of a workflow by ID.
func (a *API) workflowTriggerHandler(c *gin.Context) {
	id := c.Param("id")

	// Temporary stub logic: only "valid-id" is found, others 404
	if id != "valid-id" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "workflow not found",
		})
		return
	}

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid json body",
		})
		return
	}

	// TODO: Dispatch workflow execution to engine asynchronously.
	c.JSON(http.StatusAccepted, gin.H{
		"message": "workflow queued for execution",
		"id":      id,
	})
}
