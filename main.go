package main

import (
	"log"
	"nganime-api/config"
	"nganime-api/models"
	"nganime-api/routes"
	"nganime-api/services"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file. Coba di folder kerja dulu, lalu di folder tempat binary berada,
	// supaya tetap terbaca walau API dijalankan dari folder lain (mis. systemd/pm2).
	err := godotenv.Load()
	if err != nil {
		if exe, exeErr := os.Executable(); exeErr == nil {
			err = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
		}
	}
	if err != nil {
		log.Println("No .env file found or error loading it. Using environment variables.")
	}

	// JWT_SECRET wajib. Tanpa ini token tidak aman (atau semua user ter-logout tiap
	// API dijalankan dari konfigurasi berbeda), jadi lebih baik berhenti dengan jelas.
	if len(os.Getenv("JWT_SECRET")) < 16 {
		log.Fatal("JWT_SECRET belum diatur atau terlalu pendek (minimal 16 karakter). Isi di file .env, lalu jalankan ulang.")
	}

	// Connect to Database
	config.ConnectDB()

	// Auto Migrate the schema
	err = config.DB.AutoMigrate(&models.User{}, &models.Bookmark{}, &models.Comment{}, &models.WatchHistory{}, &models.Session{})
	if err != nil {
		log.Fatal("Failed to auto-migrate database schema:", err)
	}
	log.Println("Database Auto-Migration completed!")

	// Bersihkan sesi kedaluwarsa/dicabut secara berkala
	services.StartSessionCleanup(config.DB)

	// Initialize Gin engine
	r := gin.Default()

	// CORS Middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Serve static files for uploads
	r.Static("/uploads", "./uploads")

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
