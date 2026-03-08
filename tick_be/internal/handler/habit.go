package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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

func (h *HabitHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	serverTime := time.Now().UTC()

	query := h.DB.Where("user_id = ?", userID)

	if updatedAfter := r.URL.Query().Get("updated_after"); updatedAfter != "" {
		t, err := time.Parse(time.RFC3339, updatedAfter)
		if err == nil {
			query = query.Where("updated_at > ?", t)
		}
	}

	if r.URL.Query().Get("include_deleted") != "true" {
		query = query.Where("is_deleted = ?", false)
	}

	var habits []model.Habit
	if err := query.Find(&habits).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to fetch habits")
		return
	}

	result := make([]habitResponse, len(habits))
	for i, habit := range habits {
		result[i] = toHabitResponse(&habit)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"habits":      result,
		"server_time": serverTime.Format(time.RFC3339),
	})
}

func (h *HabitHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req createHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.ClientID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_CLIENT_ID", "client_id is required")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if len(req.Name) > 255 {
		respondError(w, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 255 characters")
		return
	}

	if req.FrequencyType != "daily" && req.FrequencyType != "weekly" {
		respondError(w, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_type must be 'daily' or 'weekly'")
		return
	}

	if req.FrequencyValue < 1 || req.FrequencyValue > 7 {
		respondError(w, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_value must be between 1 and 7")
		return
	}

	// Idempotency: return existing if client_id already exists
	var existing model.Habit
	if err := h.DB.Where("client_id = ?", req.ClientID).First(&existing).Error; err == nil {
		writeJSON(w, http.StatusOK, toHabitResponse(&existing))
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
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create habit")
		return
	}

	writeJSON(w, http.StatusCreated, toHabitResponse(&habit))
}

func (h *HabitHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&habit).Error; err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	var req updateHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.UpdatedAt != nil && habit.UpdatedAt.After(*req.UpdatedAt) {
		respondError(w, http.StatusConflict, "CONFLICT", "Server version is newer. Please re-fetch and try again.")
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if req.Name != nil {
		if *req.Name == "" {
			respondError(w, http.StatusBadRequest, "MISSING_NAME", "Name cannot be empty")
			return
		}
		if len(*req.Name) > 255 {
			respondError(w, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 255 characters")
			return
		}
		updates["name"] = *req.Name
	}

	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}

	if req.FrequencyType != nil {
		if *req.FrequencyType != "daily" && *req.FrequencyType != "weekly" {
			respondError(w, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_type must be 'daily' or 'weekly'")
			return
		}
		updates["frequency_type"] = *req.FrequencyType
	}

	if req.FrequencyValue != nil {
		if *req.FrequencyValue < 1 || *req.FrequencyValue > 7 {
			respondError(w, http.StatusBadRequest, "INVALID_FREQUENCY", "frequency_value must be between 1 and 7")
			return
		}
		updates["frequency_value"] = *req.FrequencyValue
	}

	h.DB.Model(&habit).Updates(updates)
	h.DB.First(&habit, id)

	writeJSON(w, http.StatusOK, toHabitResponse(&habit))
}

func (h *HabitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&habit).Error; err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	now := time.Now().UTC()

	h.DB.Model(&habit).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	h.DB.Model(&model.HabitLog{}).Where("habit_id = ? AND user_id = ?", id, userID).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	w.WriteHeader(http.StatusNoContent)
}
