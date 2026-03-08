package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

func (h *CategoryHandler) List(c *gin.Context) {
	userID := getUserID(c)

	var categories []model.Category
	// Presets (user_id IS NULL, is_preset = true) + user's custom categories (non-deleted)
	err := h.DB.Where(
		"(is_preset = ? AND user_id IS NULL) OR (user_id = ? AND is_deleted = ?)",
		true, userID, false,
	).Find(&categories).Error

	if err != nil {
		respondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Failed to fetch categories")
		return
	}

	result := make([]categoryResponse, len(categories))
	for i, cat := range categories {
		result[i] = toCategoryResponse(&cat)
	}

	c.JSON(http.StatusOK, gin.H{"categories": result})
}

func (h *CategoryHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if len(req.Name) > 100 {
		respondError(c, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 100 characters")
		return
	}

	// Check for duplicates among user's own categories
	var count int64
	h.DB.Model(&model.Category{}).Where("user_id = ? AND name = ? AND is_deleted = ?", userID, req.Name, false).Count(&count)
	if count > 0 {
		respondError(c, http.StatusConflict, "DUPLICATE_NAME", "A category with this name already exists")
		return
	}

	cat := model.Category{
		UserID:   &userID,
		Name:     req.Name,
		IsPreset: false,
	}

	if err := h.DB.Create(&cat).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create category")
		return
	}

	c.JSON(http.StatusCreated, toCategoryResponse(&cat))
}

func (h *CategoryHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid category ID")
		return
	}

	var cat model.Category
	if err := h.DB.First(&cat, id).Error; err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Category not found")
		return
	}

	if cat.IsPreset {
		respondError(c, http.StatusForbidden, "PRESET_IMMUTABLE", "Cannot rename a preset category")
		return
	}

	if cat.UserID == nil || *cat.UserID != userID {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Category not found or not owned by user")
		return
	}

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		respondError(c, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	if len(req.Name) > 100 {
		respondError(c, http.StatusUnprocessableEntity, "NAME_TOO_LONG", "Name must not exceed 100 characters")
		return
	}

	// Check for duplicates (exclude self)
	var count int64
	h.DB.Model(&model.Category{}).Where("user_id = ? AND name = ? AND is_deleted = ? AND id != ?", userID, req.Name, false, id).Count(&count)
	if count > 0 {
		respondError(c, http.StatusConflict, "DUPLICATE_NAME", "A category with this name already exists")
		return
	}

	h.DB.Model(&cat).Updates(map[string]interface{}{
		"name":       req.Name,
		"updated_at": time.Now().UTC(),
	})

	c.JSON(http.StatusOK, toCategoryResponse(&cat))
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid category ID")
		return
	}

	var cat model.Category
	if err := h.DB.First(&cat, id).Error; err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Category not found")
		return
	}

	if cat.IsPreset {
		respondError(c, http.StatusForbidden, "PRESET_IMMUTABLE", "Cannot delete a preset category")
		return
	}

	if cat.UserID == nil || *cat.UserID != userID {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Category not found or not owned by user")
		return
	}

	now := time.Now().UTC()

	// Soft-delete the category
	h.DB.Model(&cat).Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})

	// Nullify category_id on associated habits
	h.DB.Model(&model.Habit{}).Where("category_id = ? AND user_id = ?", id, userID).Updates(map[string]interface{}{
		"category_id": nil,
		"updated_at":  now,
	})

	c.Status(http.StatusNoContent)
}
