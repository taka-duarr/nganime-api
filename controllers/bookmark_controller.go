package controllers

import (
	"net/http"
	"nganime-api/config"
	"nganime-api/models"

	"github.com/gin-gonic/gin"
)

type AddBookmarkInput struct {
	AnimeID string `json:"anime_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Poster  string `json:"poster"`
}

func AddBookmark(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input AddBookmarkInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if already bookmarked
	var existingBookmark models.Bookmark
	if err := config.DB.Where("user_id = ? AND anime_id = ?", userID, input.AnimeID).First(&existingBookmark).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Anime already bookmarked"})
		return
	}

	bookmark := models.Bookmark{
		UserID:  userID.(uint),
		AnimeID: input.AnimeID,
		Title:   input.Title,
		Poster:  input.Poster,
	}

	if err := config.DB.Create(&bookmark).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add bookmark"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Bookmark added successfully", "bookmark": bookmark})
}

func GetBookmarks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var bookmarks []models.Bookmark
	if err := config.DB.Where("user_id = ?", userID).Find(&bookmarks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmarks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookmarks": bookmarks})
}

func RemoveBookmark(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	animeID := c.Param("anime_id")

	result := config.DB.Where("user_id = ? AND anime_id = ?", userID, animeID).Delete(&models.Bookmark{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove bookmark"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bookmark removed successfully"})
}
