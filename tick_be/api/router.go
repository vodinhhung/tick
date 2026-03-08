package api

import (
	"net/http"

	"gorm.io/gorm"

	"tick/be/internal/handler"
	"tick/be/internal/middleware"
)

func SetupRouter(db *gorm.DB, jwtSecret, googleClientID string) http.Handler {
	mux := http.NewServeMux()

	authHandler := handler.NewAuthHandler(db, jwtSecret, googleClientID)
	categoryHandler := handler.NewCategoryHandler(db)
	habitHandler := handler.NewHabitHandler(db)
	habitLogHandler := handler.NewHabitLogHandler(db)
	syncHandler := handler.NewSyncHandler(db)
	statsHandler := handler.NewStatsHandler(db)

	jwt := middleware.JWTAuth(jwtSecret)
	protect := func(h http.HandlerFunc) http.Handler {
		return jwt(http.HandlerFunc(h))
	}

	// Public
	mux.HandleFunc("POST /api/v1/auth/google", authHandler.GoogleLogin)

	// Auth
	mux.Handle("POST /api/v1/auth/refresh", protect(authHandler.Refresh))

	// Categories
	mux.Handle("GET /api/v1/categories", protect(categoryHandler.List))
	mux.Handle("POST /api/v1/categories", protect(categoryHandler.Create))
	mux.Handle("PUT /api/v1/categories/{id}", protect(categoryHandler.Update))
	mux.Handle("DELETE /api/v1/categories/{id}", protect(categoryHandler.Delete))

	// Habits
	mux.Handle("GET /api/v1/habits", protect(habitHandler.List))
	mux.Handle("POST /api/v1/habits", protect(habitHandler.Create))
	mux.Handle("PUT /api/v1/habits/{id}", protect(habitHandler.Update))
	mux.Handle("DELETE /api/v1/habits/{id}", protect(habitHandler.Delete))

	// Habit logs
	mux.Handle("GET /api/v1/habits/{habit_id}/logs", protect(habitLogHandler.List))
	mux.Handle("POST /api/v1/habits/{habit_id}/logs", protect(habitLogHandler.Create))
	mux.Handle("DELETE /api/v1/habits/{habit_id}/logs/{log_id}", protect(habitLogHandler.Delete))

	// Stats
	mux.Handle("GET /api/v1/habits/{id}/stats", protect(statsHandler.GetStats))

	// Sync
	mux.Handle("POST /api/v1/sync", protect(syncHandler.Sync))

	return mux
}
