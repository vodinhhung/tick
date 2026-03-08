package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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

func (h *HabitLogHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	habitID, err := strconv.ParseUint(r.PathValue("habit_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", habitID, userID).First(&habit).Error; err != nil {
		respondError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	serverTime := time.Now().UTC()
	query := h.DB.Where("habit_id = ? AND user_id = ?", habitID, userID)

	q := r.URL.Query()
	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			query = query.Where("completed_at >= ?", t)
		}
	}

	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			query = query.Where("completed_at <= ?", t)
		}
	}

	if updatedAfter := q.Get("updated_after"); updatedAfter != "" {
		if t, err := time.Parse(time.RFC3339, updatedAfter); err == nil {
			query = query.Where("updated_at > ?", t)
		}
	}

	if q.Get("include_deleted") != "true" {
		query = query.Where("is_deleted = ?", false)
	}

	var logs []model.HabitLog
	if err := query.Order("completed_at DESC").Find(&logs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to fetch logs")
		return
	}

	result := make([]logResponse, len(logs))
	for i, l := range logs {
		result[i] = toLogResponse(&l)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":        result,
		"server_time": serverTime.Format(time.RFC3339),
	})
}

func (h *HabitLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	habitID, err := strconv.ParseUint(r.PathValue("habit_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", habitID, userID).First(&habit).Error; err != nil {
		respondError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	var req createLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.ClientID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_CLIENT_ID", "client_id is required")
		return
	}

	if req.CompletedAt.IsZero() {
		respondError(w, http.StatusBadRequest, "MISSING_COMPLETED_AT", "completed_at is required")
		return
	}

	if req.Note != nil && len(*req.Note) > 280 {
		respondError(w, http.StatusUnprocessableEntity, "NOTE_TOO_LONG", "Note must not exceed 280 characters")
		return
	}

	// Idempotency: return existing if client_id already exists
	var existing model.HabitLog
	if err := h.DB.Where("client_id = ?", req.ClientID).First(&existing).Error; err == nil {
		writeJSON(w, http.StatusOK, toLogResponse(&existing))
		return
	}

	isExtra := h.computeIsExtra(&habit, req.CompletedAt, userID)

	entry := model.HabitLog{
		UserID:      userID,
		HabitID:     uint(habitID),
		ClientID:    req.ClientID,
		CompletedAt: req.CompletedAt,
		Note:        req.Note,
		IsExtra:     isExtra,
	}

	if err := h.DB.Create(&entry).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create log")
		return
	}

	writeJSON(w, http.StatusCreated, toLogResponse(&entry))
}

func (h *HabitLogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	habitID, err := strconv.ParseUint(r.PathValue("habit_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	logID, err := strconv.ParseUint(r.PathValue("log_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid log ID")
		return
	}

	var entry model.HabitLog
	if err := h.DB.Where("id = ? AND habit_id = ? AND user_id = ?", logID, habitID, userID).First(&entry).Error; err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Log not found or not owned by user")
		return
	}

	now := time.Now().UTC()
	h.DB.Model(&entry).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *HabitLogHandler) computeIsExtra(habit *model.Habit, completedAt time.Time, userID uint) bool {
	var count int64

	if habit.FrequencyType == "daily" {
		startOfDay := time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour)
		h.DB.Model(&model.HabitLog{}).Where(
			"habit_id = ? AND user_id = ? AND completed_at >= ? AND completed_at < ? AND is_deleted = ?",
			habit.ID, userID, startOfDay, endOfDay, false,
		).Count(&count)
		return count >= int64(habit.FrequencyValue)
	}

	if habit.FrequencyType == "weekly" {
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
