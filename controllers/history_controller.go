package controllers

import (
	"net/http"
	"nganime-api/config"
	"nganime-api/models"

	"github.com/gin-gonic/gin"
)

type MarkHistoryInput struct {
	AnimeID   string `json:"anime_id" binding:"required"`
	EpisodeID string `json:"episode_id" binding:"required"`
}

// MarkAsWatched adds an episode to the user's watch history
func MarkAsWatched(c *gin.Context) {
	// Get user ID from middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input MarkHistoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	history := models.WatchHistory{
		UserID:    userID.(uint),
		AnimeID:   input.AnimeID,
		EpisodeID: input.EpisodeID,
	}

	// Use FirstOrCreate to prevent duplicates based on user_id, anime_id, and episode_id
	result := config.DB.Where(models.WatchHistory{
		UserID:    userID.(uint),
		AnimeID:   input.AnimeID,
		EpisodeID: input.EpisodeID,
	}).FirstOrCreate(&history)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save watch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Episode marked as watched",
		"history": history,
	})
}

// GetWatchedEpisodes returns a list of watched episodes for a specific anime
func GetWatchedEpisodes(c *gin.Context) {
	// Get user ID from middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	animeID := c.Param("anime_id")
	if animeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Anime ID is required"})
		return
	}

	var histories []models.WatchHistory
	if err := config.DB.Where("user_id = ? AND anime_id = ?", userID, animeID).Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch history"})
		return
	}

	// Extract just the episode IDs for easier frontend consumption
	var watchedEpisodes []string
	for _, h := range histories {
		watchedEpisodes = append(watchedEpisodes, h.EpisodeID)
	}

	// If it's nil, return an empty array instead of null
	if watchedEpisodes == nil {
		watchedEpisodes = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"anime_id":         animeID,
		"watched_episodes": watchedEpisodes,
	})
}
