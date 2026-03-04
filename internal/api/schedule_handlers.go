package api

import (
	"encoding/json"
	"net/http"

	db "github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/devaldrete/dotbrain/internal/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateScheduleRequest struct {
	CronExpr string         `json:"cron_expr" binding:"required"`
	Payload  map[string]any `json:"payload"`
}

type UpdateScheduleRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// createScheduleHandler creates a new schedule for a workflow.
func (a *API) createScheduleHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var pgWorkflowID pgtype.UUID
	pgWorkflowID.Bytes = parsedID
	pgWorkflowID.Valid = true

	// Verify workflow exists
	_, err = a.queries.GetWorkflow(c, pgWorkflowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate cron expression
	if err := scheduler.ValidateCronExpr(req.CronExpr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedID, _ := uuid.NewV7()
	var pgSchedID pgtype.UUID
	pgSchedID.Bytes = schedID
	pgSchedID.Valid = true

	payloadBytes, _ := json.Marshal(req.Payload)
	if req.Payload == nil {
		payloadBytes = []byte("{}")
	}

	sched, err := a.queries.CreateSchedule(c, db.CreateScheduleParams{
		ID:         pgSchedID,
		WorkflowID: pgWorkflowID,
		CronExpr:   req.CronExpr,
		Payload:    payloadBytes,
		Enabled:    true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule: " + err.Error()})
		return
	}

	// Register with the scheduler if it's running
	if a.scheduler != nil {
		if err := a.scheduler.Add(sched); err != nil {
			// Schedule was created in DB but failed to register. Log but don't fail.
			c.JSON(http.StatusCreated, gin.H{
				"schedule": sched,
				"warning":  "schedule created but failed to register with cron runner: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusCreated, sched)
}

// listSchedulesHandler lists all schedules for a workflow.
func (a *API) listSchedulesHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	schedules, err := a.queries.ListSchedulesForWorkflow(c, pgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules: " + err.Error()})
		return
	}

	if schedules == nil {
		schedules = []db.Schedule{}
	}

	c.JSON(http.StatusOK, schedules)
}

// deleteScheduleHandler deletes a schedule.
func (a *API) deleteScheduleHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	// Verify schedule exists
	_, err = a.queries.GetSchedule(c, pgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := a.queries.DeleteSchedule(c, pgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}

	// Unregister from the scheduler
	if a.scheduler != nil {
		a.scheduler.RemoveByUUID(pgID)
	}

	c.Status(http.StatusNoContent)
}

// updateScheduleHandler enables or disables a schedule.
func (a *API) updateScheduleHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current schedule to check if it exists
	existingSched, err := a.queries.GetSchedule(c, pgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	sched, err := a.queries.UpdateScheduleEnabled(c, db.UpdateScheduleEnabledParams{
		ID:      pgID,
		Enabled: *req.Enabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule"})
		return
	}

	// Update the scheduler
	if a.scheduler != nil {
		if *req.Enabled && !existingSched.Enabled {
			// Re-enable: register with cron
			_ = a.scheduler.Add(sched)
		} else if !*req.Enabled && existingSched.Enabled {
			// Disable: unregister from cron
			a.scheduler.RemoveByUUID(pgID)
		}
	}

	c.JSON(http.StatusOK, sched)
}
