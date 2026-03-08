package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"tick/be/internal/model"
)

type SyncHandler struct {
	DB *gorm.DB
}

func NewSyncHandler(db *gorm.DB) *SyncHandler {
	return &SyncHandler{DB: db}
}

type syncCategoryInput struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	IsDeleted bool      `json:"is_deleted"`
	UpdatedAt time.Time `json:"updated_at"`
}

type syncHabitInput struct {
	ClientID         string    `json:"client_id"`
	Name             string    `json:"name"`
	CategoryClientID string    `json:"category_client_id,omitempty"`
	FrequencyType    string    `json:"frequency_type"`
	FrequencyValue   int       `json:"frequency_value"`
	IsDeleted        bool      `json:"is_deleted"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type syncLogInput struct {
	ClientID      string    `json:"client_id"`
	HabitClientID string    `json:"habit_client_id"`
	CompletedAt   time.Time `json:"completed_at"`
	Note          *string   `json:"note,omitempty"`
	IsDeleted     bool      `json:"is_deleted"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type syncRequest struct {
	LastSyncedAt *time.Time          `json:"last_synced_at"`
	Categories   []syncCategoryInput `json:"categories"`
	Habits       []syncHabitInput    `json:"habits"`
	Logs         []syncLogInput      `json:"logs"`
}

type syncIDMap struct {
	Categories map[string]uint `json:"categories"`
	Habits     map[string]uint `json:"habits"`
	Logs       map[string]uint `json:"logs"`
}

type syncResponse struct {
	ServerTime string             `json:"server_time"`
	Categories []categoryResponse `json:"categories"`
	Habits     []habitResponse    `json:"habits"`
	Logs       []logResponse      `json:"logs"`
	IDMap      syncIDMap          `json:"id_map"`
}

func (h *SyncHandler) Sync(c *gin.Context) {
	userID := getUserID(c)
	serverTime := time.Now().UTC()

	var req syncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	idMap := syncIDMap{
		Categories: make(map[string]uint),
		Habits:     make(map[string]uint),
		Logs:       make(map[string]uint),
	}

	// Process pushed categories
	for _, catInput := range req.Categories {
		if catInput.ClientID == "" {
			continue
		}

		// Find existing by a synthetic client_id approach. Categories don't have client_id in DB,
		// so we match by name + user_id for custom categories. For sync we'll use a convention:
		// categories are matched by name for the user.
		var existing model.Category
		err := h.DB.Where("user_id = ? AND name = ?", userID, catInput.Name).First(&existing).Error

		if err == nil {
			// Exists on server: server wins if server is newer
			if !existing.UpdatedAt.After(catInput.UpdatedAt) {
				h.DB.Model(&existing).Updates(map[string]interface{}{
					"name":       catInput.Name,
					"is_deleted": catInput.IsDeleted,
					"updated_at": serverTime,
				})
			}
			idMap.Categories[catInput.ClientID] = existing.ID
		} else {
			// Create new
			cat := model.Category{
				UserID:    &userID,
				Name:      catInput.Name,
				IsPreset:  false,
				IsDeleted: catInput.IsDeleted,
			}
			if err := h.DB.Create(&cat).Error; err == nil {
				idMap.Categories[catInput.ClientID] = cat.ID
			}
		}
	}

	// Process pushed habits
	for _, habitInput := range req.Habits {
		if habitInput.ClientID == "" {
			continue
		}

		var existing model.Habit
		err := h.DB.Where("client_id = ?", habitInput.ClientID).First(&existing).Error

		if err == nil {
			// Exists: server wins if server is newer
			if !existing.UpdatedAt.After(habitInput.UpdatedAt) {
				updates := map[string]interface{}{
					"name":            habitInput.Name,
					"frequency_type":  habitInput.FrequencyType,
					"frequency_value": habitInput.FrequencyValue,
					"is_deleted":      habitInput.IsDeleted,
					"updated_at":      serverTime,
				}
				// Resolve category_client_id to server category ID
				if habitInput.CategoryClientID != "" {
					if catID, ok := idMap.Categories[habitInput.CategoryClientID]; ok {
						updates["category_id"] = catID
					}
				}
				h.DB.Model(&existing).Updates(updates)
			}
			idMap.Habits[habitInput.ClientID] = existing.ID
		} else {
			// Create new
			habit := model.Habit{
				UserID:         userID,
				ClientID:       habitInput.ClientID,
				Name:           habitInput.Name,
				FrequencyType:  habitInput.FrequencyType,
				FrequencyValue: habitInput.FrequencyValue,
				IsDeleted:      habitInput.IsDeleted,
			}
			if habitInput.CategoryClientID != "" {
				if catID, ok := idMap.Categories[habitInput.CategoryClientID]; ok {
					habit.CategoryID = &catID
				}
			}
			if err := h.DB.Create(&habit).Error; err == nil {
				idMap.Habits[habitInput.ClientID] = habit.ID
			}
		}
	}

	// Process pushed logs
	for _, logInput := range req.Logs {
		if logInput.ClientID == "" {
			continue
		}

		var existing model.HabitLog
		err := h.DB.Where("client_id = ?", logInput.ClientID).First(&existing).Error

		if err == nil {
			// Exists: server wins if server is newer
			if !existing.UpdatedAt.After(logInput.UpdatedAt) {
				updates := map[string]interface{}{
					"completed_at": logInput.CompletedAt,
					"note":         logInput.Note,
					"is_deleted":   logInput.IsDeleted,
					"updated_at":   serverTime,
				}
				h.DB.Model(&existing).Updates(updates)
			}
			idMap.Logs[logInput.ClientID] = existing.ID
		} else {
			// Resolve habit_client_id to server habit ID
			var habitID uint
			if logInput.HabitClientID != "" {
				if hID, ok := idMap.Habits[logInput.HabitClientID]; ok {
					habitID = hID
				} else {
					// Try to find by client_id in DB
					var habit model.Habit
					if err := h.DB.Where("client_id = ? AND user_id = ?", logInput.HabitClientID, userID).First(&habit).Error; err == nil {
						habitID = habit.ID
					}
				}
			}

			if habitID == 0 {
				continue // Skip logs without a valid habit
			}

			logEntry := model.HabitLog{
				UserID:      userID,
				HabitID:     habitID,
				ClientID:    logInput.ClientID,
				CompletedAt: logInput.CompletedAt,
				Note:        logInput.Note,
				IsDeleted:   logInput.IsDeleted,
			}
			if err := h.DB.Create(&logEntry).Error; err == nil {
				idMap.Logs[logInput.ClientID] = logEntry.ID
			}
		}
	}

	// Pull server-side changes since last_synced_at
	var changedCategories []model.Category
	var changedHabits []model.Habit
	var changedLogs []model.HabitLog

	catQuery := h.DB.Where("(is_preset = ? AND user_id IS NULL) OR user_id = ?", true, userID)
	habitQuery := h.DB.Where("user_id = ?", userID)
	logQuery := h.DB.Where("user_id = ?", userID)

	if req.LastSyncedAt != nil {
		catQuery = catQuery.Where("updated_at > ?", *req.LastSyncedAt)
		habitQuery = habitQuery.Where("updated_at > ?", *req.LastSyncedAt)
		logQuery = logQuery.Where("updated_at > ?", *req.LastSyncedAt)
	}

	catQuery.Find(&changedCategories)
	habitQuery.Find(&changedHabits)
	logQuery.Find(&changedLogs)

	// Build response
	catResponses := make([]categoryResponse, len(changedCategories))
	for i, cat := range changedCategories {
		catResponses[i] = toCategoryResponse(&cat)
	}

	habitResponses := make([]habitResponse, len(changedHabits))
	for i, habit := range changedHabits {
		habitResponses[i] = toHabitResponse(&habit)
	}

	logResponses := make([]logResponse, len(changedLogs))
	for i, log := range changedLogs {
		logResponses[i] = toLogResponse(&log)
	}

	c.JSON(http.StatusOK, syncResponse{
		ServerTime: serverTime.Format(time.RFC3339),
		Categories: catResponses,
		Habits:     habitResponses,
		Logs:       logResponses,
		IDMap:      idMap,
	})
}
