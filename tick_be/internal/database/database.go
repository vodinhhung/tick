package database

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"tick/be/config"
	"tick/be/internal/model"
)

func InitDB(cfg *config.DatabaseConfig) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&model.User{}, &model.Category{}, &model.Habit{}, &model.HabitLog{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	seedPresetCategories(db)

	fmt.Println("Database connection established and migration complete.")
	return db
}

func seedPresetCategories(db *gorm.DB) {
	presets := []string{"Health", "Fitness", "Mindfulness", "Learning", "Productivity", "Other"}

	for _, name := range presets {
		var count int64
		db.Model(&model.Category{}).Where("name = ? AND is_preset = ?", name, true).Count(&count)
		if count == 0 {
			cat := model.Category{
				Name:     name,
				IsPreset: true,
				UserID:   nil,
			}
			if err := db.Create(&cat).Error; err != nil {
				log.Printf("Failed to seed preset category %s: %v", name, err)
			}
		}
	}
}
