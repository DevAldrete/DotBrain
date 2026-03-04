package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/devaldrete/dotbrain/internal/core"
	"github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewAPI(pool *pgxpool.Pool) *API {
	var queries *db.Queries
	if pool != nil {
		queries = db.New(pool)
	}
	return &API{
		pool:    pool,
		queries: queries,
	}
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

		// Workflow Endpoints
		v1.POST("/workflows", a.createWorkflowHandler)
		v1.GET("/workflows", a.listWorkflowsHandler)
		v1.GET("/workflows/:id", a.getWorkflowHandler)
		v1.PUT("/workflows/:id", a.updateWorkflowHandler)
		v1.DELETE("/workflows/:id", a.deleteWorkflowHandler)
		v1.POST("/workflows/:id/trigger", a.workflowTriggerHandler)
		v1.GET("/workflows/:id/runs", a.listWorkflowRunsHandler)

		// Run Endpoints
		v1.GET("/runs/:id", a.getRunHandler)
		v1.GET("/runs/:id/nodes", a.listNodeExecutionsHandler)
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
	if err := a.pool.Ping(c); err != nil {
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
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	workflow, err := a.queries.GetWorkflow(c, pgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	// Create a workflow run
	runID, _ := uuid.NewV7()
	var pgRunID pgtype.UUID
	pgRunID.Bytes = runID
	pgRunID.Valid = true

	inputBytes, _ := json.Marshal(payload)

	_, err = a.queries.CreateWorkflowRun(c, db.CreateWorkflowRunParams{
		ID:         pgRunID,
		WorkflowID: pgID,
		Status:     "pending",
		InputData:  inputBytes,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workflow run"})
		return
	}

	// Run workflow asynchronously
	go func(runID pgtype.UUID, w db.Workflow, initialData map[string]any) {
		ctx := context.Background()

		// Transition to "running" with started_at
		a.transitionToRunning(ctx, runID)

		// Setup Engine
		engine := core.NewEngine()
		engine.Hook = NewDBNodeHook(a.queries, runID)

		def, err := core.ParseDefinition(w.Definition)
		if err != nil {
			a.updateRunStatus(ctx, runID, "failed", nil, err.Error())
			return
		}

		if err := engine.LoadFromDefinition(def); err != nil {
			a.updateRunStatus(ctx, runID, "failed", nil, err.Error())
			return
		}

		output, err := engine.Execute(ctx, initialData)
		if err != nil {
			a.updateRunStatus(ctx, runID, "failed", nil, err.Error())
			return
		}

		a.updateRunStatus(ctx, runID, "completed", output, "")
	}(pgRunID, workflow, payload)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "workflow queued for execution",
		"run_id":  runID.String(),
	})
}

// transitionToRunning transitions a workflow run from "pending" to "running"
// and sets started_at to the current time.
func (a *API) transitionToRunning(ctx context.Context, id pgtype.UUID) {
	now := time.Now()
	var pgNow pgtype.Timestamptz
	pgNow.Time = now
	pgNow.Valid = true

	a.queries.UpdateWorkflowRunStatus(ctx, db.UpdateWorkflowRunStatusParams{
		ID:        id,
		Status:    "running",
		StartedAt: pgNow,
	})
}

// updateRunStatus updates a workflow run to a terminal state.
func (a *API) updateRunStatus(ctx context.Context, id pgtype.UUID, status string, output map[string]any, errMsg string) {
	var outputBytes []byte
	if output != nil {
		outputBytes, _ = json.Marshal(output)
	}

	var pgErr pgtype.Text
	if errMsg != "" {
		pgErr.String = errMsg
		pgErr.Valid = true
	}

	now := time.Now()
	var pgNow pgtype.Timestamptz
	pgNow.Time = now
	pgNow.Valid = true

	a.queries.UpdateWorkflowRunStatus(ctx, db.UpdateWorkflowRunStatusParams{
		ID:          id,
		Status:      status,
		OutputData:  outputBytes,
		Error:       pgErr,
		CompletedAt: pgNow,
	})
}

type CreateWorkflowRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Definition  any    `json:"definition" binding:"required"`
}

func (a *API) createWorkflowHandler(c *gin.Context) {
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defBytes, err := json.Marshal(req.Definition)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid definition format"})
		return
	}

	// UUID v7 for newer better DB locality
	id, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate id"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = id
	pgID.Valid = true

	workflow, err := a.queries.CreateWorkflow(c, db.CreateWorkflowParams{
		ID:          pgID,
		Name:        req.Name,
		Description: req.Description,
		Definition:  defBytes,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workflow: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

func (a *API) listWorkflowsHandler(c *gin.Context) {
	workflows, err := a.queries.ListWorkflows(c, db.ListWorkflowsParams{
		Limit:  100,
		Offset: 0,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workflows: " + err.Error()})
		return
	}

	// If no workflows, return empty array instead of null
	if workflows == nil {
		workflows = []db.Workflow{}
	}

	c.JSON(http.StatusOK, workflows)
}

func (a *API) getWorkflowHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	workflow, err := a.queries.GetWorkflow(c, pgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

func (a *API) updateWorkflowHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defBytes, err := json.Marshal(req.Definition)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid definition format"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	workflow, err := a.queries.UpdateWorkflow(c, db.UpdateWorkflowParams{
		ID:          pgID,
		Name:        req.Name,
		Description: req.Description,
		Definition:  defBytes,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update workflow"})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

func (a *API) deleteWorkflowHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	_, err = a.queries.GetWorkflow(c, pgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}

	if err := a.queries.DeleteWorkflow(c, pgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete workflow"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (a *API) listWorkflowRunsHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	runs, err := a.queries.ListWorkflowRuns(c, db.ListWorkflowRunsParams{
		WorkflowID: pgID,
		Limit:      100,
		Offset:     0,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list runs: " + err.Error()})
		return
	}

	if runs == nil {
		runs = []db.WorkflowRun{}
	}

	c.JSON(http.StatusOK, runs)
}

func (a *API) getRunHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	run, err := a.queries.GetWorkflowRun(c, pgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func (a *API) listNodeExecutionsHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	executions, err := a.queries.ListNodeExecutionsForRun(c, pgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list node executions: " + err.Error()})
		return
	}

	if executions == nil {
		executions = []db.NodeExecution{}
	}

	c.JSON(http.StatusOK, executions)
}
