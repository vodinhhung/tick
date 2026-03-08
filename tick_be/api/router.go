package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"tick/be/internal/handler"
	"tick/be/internal/middleware"
)

func SetupRouter(db *gorm.DB, jwtSecret, googleClientID string) *gin.Engine {
	r := gin.Default()

	authHandler := handler.NewAuthHandler(db, jwtSecret, googleClientID)
	categoryHandler := handler.NewCategoryHandler(db)
	habitHandler := handler.NewHabitHandler(db)
	habitLogHandler := handler.NewHabitLogHandler(db)
	syncHandler := handler.NewSyncHandler(db)
	statsHandler := handler.NewStatsHandler(db)

	v1 := r.Group("/api/v1")
	{
		// Public routes
		v1.POST("/auth/google", authHandler.GoogleLogin)

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(jwtSecret))
		{
			// Auth
			protected.POST("/auth/refresh", authHandler.Refresh)

			// Categories
			protected.GET("/categories", categoryHandler.List)
			protected.POST("/categories", categoryHandler.Create)
			protected.PUT("/categories/:id", categoryHandler.Update)
			protected.DELETE("/categories/:id", categoryHandler.Delete)

			// Habits
			protected.GET("/habits", habitHandler.List)
			protected.POST("/habits", habitHandler.Create)
			protected.PUT("/habits/:id", habitHandler.Update)
			protected.DELETE("/habits/:id", habitHandler.Delete)

			// Habit Logs
			protected.GET("/habits/:habit_id/logs", habitLogHandler.List)
			protected.POST("/habits/:habit_id/logs", habitLogHandler.Create)
			protected.DELETE("/habits/:habit_id/logs/:log_id", habitLogHandler.Delete)

			// Sync
			protected.POST("/sync", syncHandler.Sync)

			// Stats
			protected.GET("/habits/:id/stats", statsHandler.GetStats)
		}
	}

	return r
}
