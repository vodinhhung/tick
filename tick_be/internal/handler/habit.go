package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"tick/be/internal/model"
)

type HabitHandler struct {
	DB *gorm.DB
}

func NewHabitHandler(db *gorm.DB) *HabitHandler {
	return &HabitHandler{DB: db}
}

type createHabitRequest struct {
	ClientID       string `json:"client_id"`
	Name           string `json:"name"`
	CategoryID     *uint  `json:"category_id,omitempty"`
	FrequencyType  string `json:"frequency_type"`
	FrequencyValue int    `json:"frequency_value"`
}

type updateHabitRequest struct {
	Name           *string    `json:"name,omitempty"`
	CategoryID     *uint      `json:"category_id,omitempty"`
	FrequencyType  *string    `json:"frequency_type,omitempty"`
	FrequencyValue *int       `json:"frequency_value,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

type habitResponse struct {
	ID             uint      `json:"id"`
	ClientID       string    `json:"client_id"`
	Name           string    `json:"name"`
	CategoryID     *uint     `json:"category_id,omitempty"`
	FrequencyType  string    `json:"frequency_type"`
	FrequencyValue int       `json:"frequency_value"`
	IsDeleted      bool      `json:"is_deleted"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func toHabitResponse(h *model.Habit) habitResponse {
	return habitResponse{
		ID:             h.ID,
		ClientID:       h.ClientID,
		Name:           h.Name,
		CategoryID:     h.CategoryID,
		FrequencyType:  h.FrequencyType,
		FrequencyValue: h.FrequencyValue,
		IsDeleted:      h.IsDeleted,
		CreatedAt:      h.CreatedAt,
		UpdatedAt:      h.UpdatedAt,
	}
}

func (h *HabitHandler) List(c *gin.Context) {
	userID := getUserID(c)
	serverTime := time.Now().UTC()

	query := h.DB.Where("user_id = ?", userID)

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

	var habits []model.Habit
	if err := query.Find(&habits).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Failed to fetch habits")
		return
	}

	result := make([]habitResponse, len(habits))
	for i, habit := range habits {
		result[i] = toHabitResponse(&habit)
	}

	c.JSON(http.StatusOK, gin.H{
		"habits":      result,
		"server_time": serverTime.Format(time.RFC3339),
	})
}

func (h *HabitHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	var req createHabitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.ClientID == "" {
		respondError(c, http.StatusBadRequest, "MISSING_CLIENT_ID", "client_id is required")
		return
	}

	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if len(req.Name) > 255 {
		respondError(c, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 255 characters")
		return
	}

	if req.FrequencyType != "daily" && req.FrequencyType != "weekly" {
		respondError(c, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_type must be 'daily' or 'weekly'")
		return
	}

	if req.FrequencyValue < 1 || req.FrequencyValue > 7 {
		respondError(c, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_value must be between 1 and 7")
		return
	}

	// Idempotency: check if client_id already exists
	var existing model.Habit
	if err := h.DB.Where("client_id = ?", req.ClientID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, toHabitResponse(&existing))
		return
	}

	habit := model.Habit{
		UserID:         userID,
		ClientID:       req.ClientID,
		Name:           req.Name,
		CategoryID:     req.CategoryID,
		FrequencyType:  req.FrequencyType,
		FrequencyValue: req.FrequencyValue,
	}

	if err := h.DB.Create(&habit).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create habit")
		return
	}

	c.JSON(http.StatusCreated, toHabitResponse(&habit))
}

func (h *HabitHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&habit).Error; err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	var req updateHabitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Optimistic locking
	if req.UpdatedAt != nil && habit.UpdatedAt.After(*req.UpdatedAt) {
		respondError(c, http.StatusConflict, "CONFLICT", "Server version is newer. Please re-fetch and try again.")
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if req.Name != nil {
		if *req.Name == "" {
			respondError(c, http.StatusBadRequest, "MISSING_NAME", "Name cannot be empty")
			return
		}
		if len(*req.Name) > 255 {
			respondError(c, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 255 characters")
			return
		}
		updates["name"] = *req.Name
	}

	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}

	if req.FrequencyType != nil {
		if *req.FrequencyType != "daily" && *req.FrequencyType != "weekly" {
			respondError(c, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_type must be 'daily' or 'weekly'")
			return
		}
		updates["frequency_type"] = *req.FrequencyType
	}

	if req.FrequencyValue != nil {
		if *req.FrequencyValue < 1 || *req.FrequencyValue > 7 {
			respondError(c, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_value must be between 1 and 7")
			return
		}
		updates["frequency_value"] = *req.FrequencyValue
	}

	h.DB.Model(&habit).Updates(updates)

	// Reload to get updated values
	h.DB.First(&habit, id)

	c.JSON(http.StatusOK, toHabitResponse(&habit))
}

func (h *HabitHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&habit).Error; err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	now := time.Now().UTC()

	// Soft-delete the habit
	h.DB.Model(&habit).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	// Cascade soft-delete to logs
	h.DB.Model(&model.HabitLog{}).Where("habit_id = ? AND user_id = ?", id, userID).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	c.Status(http.StatusNoContent)
}
