package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"wfagent/internal/base/response"
)

type submitRequest struct {
	TaskType string         `json:"task_type" binding:"required"`
	Payload  map[string]any `json:"payload"`
}

type Handler struct {
	broker Broker
}

func NewHandler(db *gorm.DB, redisClient *redis.Client) *Handler {
	return &Handler{broker: NewBroker(db, redisClient)}
}

// RegisterRoutes registers async task endpoints backed by PostgreSQL and Redis.
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client) {
	handler := NewHandler(db, redisClient)
	group := v1.Group("/tasks")
	group.POST("", handler.Submit)
	group.GET("/:task_id", handler.Query)
	group.GET("/:task_id/events", handler.Events)
	group.POST("/:task_id/cancel", handler.Cancel)
}

// Submit godoc
// @Summary Submit async task
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body submitRequest true "Task request"
// @Success 201 {object} response.Body{data=task.AsyncTask}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/tasks [post]
func (h *Handler) Submit(c *gin.Context) {
	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	record, err := h.broker.Submit(c.Request.Context(), req.TaskType, req.Payload)
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Created(c, record)
}

// Query godoc
// @Summary Query async task
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param task_id path string true "Task ID"
// @Success 200 {object} response.Body{data=task.AsyncTask}
// @Failure 401 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /v1/tasks/{task_id} [get]
func (h *Handler) Query(c *gin.Context) {
	record, err := h.broker.Query(c.Request.Context(), c.Param("task_id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, response.ErrNotFound)
		return
	}
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, record)
}

// Events godoc
// @Summary Stream async task progress
// @Tags Tasks
// @Produce text/event-stream
// @Security BearerAuth
// @Param task_id path string true "Task ID"
// @Success 200 {string} string "SSE stream"
// @Failure 401 {object} response.Body
// @Router /v1/tasks/{task_id}/events [get]
func (h *Handler) Events(c *gin.Context) {
	taskID := c.Param("task_id")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Stream(func(w io.Writer) bool {
		record, err := h.broker.Query(c.Request.Context(), taskID)
		if err != nil {
			_, _ = fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
			return false
		}
		data, _ := json.Marshal(record)
		_, _ = fmt.Fprintf(w, "event: progress\ndata: %s\n\n", data)
		if record.Status == "success" || record.Status == "failed" || record.Status == "cancelled" {
			return false
		}
		time.Sleep(time.Second)
		return true
	})
}

// Cancel godoc
// @Summary Cancel async task
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param task_id path string true "Task ID"
// @Success 200 {object} response.Body{data=task.AsyncTask}
// @Failure 401 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /v1/tasks/{task_id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	record, err := h.broker.Cancel(c.Request.Context(), c.Param("task_id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, response.ErrNotFound)
		return
	}
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, record)
}
