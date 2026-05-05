package main

import (
	"log"
	"nganime-api/config"
	"nganime-api/models"
	"nganime-api/routes"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading it. Using environment variables.")
	}

	// Connect to Database
	config.ConnectDB()

	// Auto Migrate the schema
	err = config.DB.AutoMigrate(&models.User{}, &models.Bookmark{})
	if err != nil {
		log.Fatal("Failed to auto-migrate database schema:", err)
	}
	log.Println("Database Auto-Migration completed!")

	// Initialize Gin engine
	r := gin.Default()

	// Setup Routes
	routes.SetupRoutes(r)

	// Get port from env or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the server
	log.Printf("Server is running on port %s", port)
	r.Run(":" + port)
}
