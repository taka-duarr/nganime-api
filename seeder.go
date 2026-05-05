package main

import (
	"log"
	"nganime-api/config"
	"nganime-api/models"
	"nganime-api/utils"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it")
	}

	// Connect to Database
	config.ConnectDB()

	// 1. Create Dummy Users
	hashedPassword1, _ := utils.HashPassword("password123")
	hashedPassword2, _ := utils.HashPassword("rahasia321")

	users := []models.User{
		{Username: "wibu_sejati", Password: hashedPassword1},
		{Username: "anime_lovers", Password: hashedPassword2},
	}

	for i := range users {
		// Use FirstOrCreate so it doesn't duplicate if run multiple times
		config.DB.Where("username = ?", users[i].Username).FirstOrCreate(&users[i])
	}

	// Fetch users to get their IDs
	var user1, user2 models.User
	config.DB.Where("username = ?", "wibu_sejati").First(&user1)
	config.DB.Where("username = ?", "anime_lovers").First(&user2)

	// 2. Create Dummy Bookmarks
	bookmarks := []models.Bookmark{
		{UserID: user1.ID, AnimeID: "naruto", Title: "Naruto", Poster: "https://cdn.myanimelist.net/images/anime/13/17405.jpg"},
		{UserID: user1.ID, AnimeID: "one-piece", Title: "One Piece", Poster: "https://cdn.myanimelist.net/images/anime/6/73245.jpg"},
		{UserID: user2.ID, AnimeID: "shingeki-no-kyojin", Title: "Attack on Titan", Poster: "https://cdn.myanimelist.net/images/anime/10/47347.jpg"},
		{UserID: user2.ID, AnimeID: "kimetsu-no-yaiba", Title: "Demon Slayer", Poster: "https://cdn.myanimelist.net/images/anime/1286/99889.jpg"},
	}

	for i := range bookmarks {
		config.DB.Where("user_id = ? AND anime_id = ?", bookmarks[i].UserID, bookmarks[i].AnimeID).FirstOrCreate(&bookmarks[i])
	}

	log.Println("✅ Data dummy berhasil ditambahkan ke database!")
}
