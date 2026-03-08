package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"tick/be/internal/model"
)

type CategoryHandler struct {
	DB *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{DB: db}
}

type categoryRequest struct {
	Name string `json:"name"`
}

type categoryResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	IsPreset  bool      `json:"is_preset"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toCategoryResponse(cat *model.Category) categoryResponse {
	return categoryResponse{
		ID:        cat.ID,
		Name:      cat.Name,
		IsPreset:  cat.IsPreset,
		IsDeleted: cat.IsDeleted,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var categories []model.Category
	err := h.DB.Where(
		"(is_preset = ? AND user_id IS NULL) OR (user_id = ? AND is_deleted = ?)",
		true, userID, false,
	).Find(&categories).Error

	if err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to fetch categories")
		return
	}

	result := make([]categoryResponse, len(categories))
	for i, cat := range categories {
		result[i] = toCategoryResponse(&cat)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"categories": result})
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if len(req.Name) > 100 {
		respondError(w, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 100 characters")
		return
	}

	var count int64
	h.DB.Model(&model.Category{}).Where("user_id = ? AND name = ? AND is_deleted = ?", userID, req.Name, false).Count(&count)
	if count > 0 {
		respondError(w, http.StatusConflict, "DUPLICATE_NAME", "A category with this name already exists")
		return
	}

	cat := model.Category{
		UserID:   &userID,
		Name:     req.Name,
		IsPreset: false,
	}

	if err := h.DB.Create(&cat).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create category")
		return
	}

	writeJSON(w, http.StatusCreated, toCategoryResponse(&cat))
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID")
		return
	}

	var cat model.Category
	if err := h.DB.First(&cat, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Category not found")
		return
	}

	if cat.IsPreset {
		respondError(w, http.StatusForbidden, "PRESET_IMMUTABLE", "Cannot rename a preset category")
		return
	}

	if cat.UserID == nil || *cat.UserID != userID {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Category not found or not owned by user")
		return
	}

	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		respondError(w, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if len(req.Name) > 100 {
		respondError(w, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 100 characters")
		return
	}

	var count int64
	h.DB.Model(&model.Category{}).Where("user_id = ? AND name = ? AND is_deleted = ? AND id != ?", userID, req.Name, false, id).Count(&count)
	if count > 0 {
		respondError(w, http.StatusConflict, "DUPLICATE_NAME", "A category with this name already exists")
		return
	}

	h.DB.Model(&cat).Updates(map[string]interface{}{
		"name":       req.Name,
		"updated_at": time.Now().UTC(),
	})

	writeJSON(w, http.StatusOK, toCategoryResponse(&cat))
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID")
		return
	}

	var cat model.Category
	if err := h.DB.First(&cat, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Category not found")
		return
	}

	if cat.IsPreset {
		respondError(w, http.StatusForbidden, "PRESET_IMMUTABLE", "Cannot delete a preset category")
		return
	}

	if cat.UserID == nil || *cat.UserID != userID {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Category not found or not owned by user")
		return
	}

	now := time.Now().UTC()

	h.DB.Model(&cat).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	h.DB.Model(&model.Habit{}).Where("category_id = ? AND user_id = ?", id, userID).Updates(map[string]interface{}{
		"category_id": nil,
		"updated_at":  now,
	})

	w.WriteHeader(http.StatusNoContent)
}
