package main

import (
	"fmt"
	"log"

	"tick/be/api"
	"tick/be/config"
	"tick/be/internal/database"
)

func main() {
	cfg, err := config.LoadConfig("config/cfg")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db := database.InitDB(&cfg.Database)

	router := api.SetupRouter(db, cfg.JWTSecret, cfg.GoogleClientID)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("Server starting at http://localhost%s\n", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
