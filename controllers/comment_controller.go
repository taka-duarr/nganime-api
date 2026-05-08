package controllers

import (
	"net/http"
	"nganime-api/config"
	"nganime-api/models"

	"github.com/gin-gonic/gin"
)

type AddCommentInput struct {
	AnimeID string `json:"anime_id" binding:"required"`
	Content string `json:"content" binding:"required"`
}

func AddComment(c *gin.Context) {
	// Must be logged in
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input AddCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := models.Comment{
		UserID:  userID.(uint),
		AnimeID: input.AnimeID,
		Content: input.Content,
	}

	if err := config.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}

	// Preload User to return details
	config.DB.Preload("User").First(&comment, comment.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Comment added successfully",
		"comment": gin.H{
			"id":         comment.ID,
			"anime_id":   comment.AnimeID,
			"content":    comment.Content,
			"created_at": comment.CreatedAt,
			"user": gin.H{
				"id":              comment.User.ID,
				"username":        comment.User.Username,
				"profile_picture": comment.User.ProfilePicture,
			},
		},
	})
}

func GetCommentsByAnime(c *gin.Context) {
	animeID := c.Param("anime_id")

	var comments []models.Comment
	// Fetch comments and include user details. Order by newest first.
	if err := config.DB.Preload("User").Where("anime_id = ?", animeID).Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}

	// Format response to omit sensitive user data
	var response []gin.H
	for _, comment := range comments {
		response = append(response, gin.H{
			"id":         comment.ID,
			"anime_id":   comment.AnimeID,
			"content":    comment.Content,
			"created_at": comment.CreatedAt,
			"user": gin.H{
				"id":              comment.User.ID,
				"username":        comment.User.Username,
				"profile_picture": comment.User.ProfilePicture,
			},
		})
	}

	// If no comments, return empty array instead of null
	if response == nil {
		response = make([]gin.H, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": response,
	})
}
