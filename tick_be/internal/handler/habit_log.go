package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"tick/be/internal/model"
)

type HabitLogHandler struct {
	DB *gorm.DB
}

func NewHabitLogHandler(db *gorm.DB) *HabitLogHandler {
	return &HabitLogHandler{DB: db}
}

type createLogRequest struct {
	ClientID    string    `json:"client_id"`
	CompletedAt time.Time `json:"completed_at"`
	Note        *string   `json:"note,omitempty"`
}

type logResponse struct {
	ID          uint      `json:"id"`
	ClientID    string    `json:"client_id"`
	HabitID     uint      `json:"habit_id"`
	CompletedAt time.Time `json:"completed_at"`
	Note        *string   `json:"note,omitempty"`
	IsExtra     bool      `json:"is_extra"`
	IsDeleted   bool      `json:"is_deleted"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toLogResponse(l *model.HabitLog) logResponse {
	return logResponse{
		ID:          l.ID,
		ClientID:    l.ClientID,
		HabitID:     l.HabitID,
		CompletedAt: l.CompletedAt,
		Note:        l.Note,
		IsExtra:     l.IsExtra,
		IsDeleted:   l.IsDeleted,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}

func (h *HabitLogHandler) List(c *gin.Context) {
	userID := getUserID(c)
	habitID, err := strconv.ParseUint(c.Param("habit_id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	// Verify habit ownership
	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", habitID, userID).First(&habit).Error; err != nil {
		respondError(c, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	serverTime := time.Now().UTC()
	query := h.DB.Where("habit_id = ? AND user_id = ?", habitID, userID)

	if from := c.Query("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err == nil {
			query = query.Where("completed_at >= ?", t)
		}
	}

	if to := c.Query("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err == nil {
			query = query.Where("completed_at <= ?", t)
		}
	}

	if updatedAfter := c.Query("updated_after"); updatedAfter != "" {
		t, err := time.Parse(time.RFC3339, updatedAfter)
		if err == nil {
			query = query.Where("updated_at > ?", t)
		}
	}

	includeDeleted := c.Query("include_deleted") == "true"
	if !includeDeleted {
		query = query.Where("is_deleted = ?", false)
	}

	var logs []model.HabitLog
	if err := query.Order("completed_at DESC").Find(&logs).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Failed to fetch logs")
		return
	}

	result := make([]logResponse, len(logs))
	for i, log := range logs {
		result[i] = toLogResponse(&log)
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        result,
		"server_time": serverTime.Format(time.RFC3339),
	})
}

func (h *HabitLogHandler) Create(c *gin.Context) {
	userID := getUserID(c)
	habitID, err := strconv.ParseUint(c.Param("habit_id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	// Verify habit ownership
	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", habitID, userID).First(&habit).Error; err != nil {
		respondError(c, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	var req createLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.ClientID == "" {
		respondError(c, http.StatusBadRequest, "MISSING_CLIENT_ID", "client_id is required")
		return
	}

	if req.CompletedAt.IsZero() {
		respondError(c, http.StatusBadRequest, "MISSING_COMPLETED_AT", "completed_at is required")
		return
	}

	if req.Note != nil && len(*req.Note) > 280 {
		respondError(c, http.StatusUnprocessableEntity, "NOTE_TOO_LONG", "Note must not exceed 280 characters")
		return
	}

	// Idempotency: check if client_id already exists
	var existing model.HabitLog
	if err := h.DB.Where("client_id = ?", req.ClientID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, toLogResponse(&existing))
		return
	}

	// Compute is_extra based on existing logs for the same period
	isExtra := h.computeIsExtra(&habit, req.CompletedAt, userID)

	log := model.HabitLog{
		UserID:      userID,
		HabitID:     uint(habitID),
		ClientID:    req.ClientID,
		CompletedAt: req.CompletedAt,
		Note:        req.Note,
		IsExtra:     isExtra,
	}

	if err := h.DB.Create(&log).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create log")
		return
	}

	c.JSON(http.StatusCreated, toLogResponse(&log))
}

func (h *HabitLogHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	habitID, err := strconv.ParseUint(c.Param("habit_id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	logID, err := strconv.ParseUint(c.Param("log_id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid log ID")
		return
	}

	var log model.HabitLog
	if err := h.DB.Where("id = ? AND habit_id = ? AND user_id = ?", logID, habitID, userID).First(&log).Error; err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Log not found or not owned by user")
		return
	}

	now := time.Now().UTC()
	h.DB.Model(&log).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	c.Status(http.StatusNoContent)
}

func (h *HabitLogHandler) computeIsExtra(habit *model.Habit, completedAt time.Time, userID uint) bool {
	var count int64

	if habit.FrequencyType == "daily" {
		// Count non-deleted logs for the same day
		startOfDay := time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour)
		h.DB.Model(&model.HabitLog{}).Where(
			"habit_id = ? AND user_id = ? AND completed_at >= ? AND completed_at < ? AND is_deleted = ?",
			habit.ID, userID, startOfDay, endOfDay, false,
		).Count(&count)
		return count >= int64(habit.FrequencyValue)
	} else if habit.FrequencyType == "weekly" {
		// Count non-deleted logs for the same week (Monday-based)
		weekday := completedAt.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		startOfWeek := time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day()-int(weekday)+1, 0, 0, 0, 0, time.UTC)
		endOfWeek := startOfWeek.Add(7 * 24 * time.Hour)
		h.DB.Model(&model.HabitLog{}).Where(
			"habit_id = ? AND user_id = ? AND completed_at >= ? AND completed_at < ? AND is_deleted = ?",
			habit.ID, userID, startOfWeek, endOfWeek, false,
		).Count(&count)
		return count >= int64(habit.FrequencyValue)
	}

	return false
}
