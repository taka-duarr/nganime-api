package controllers

import (
	"net/http"
	"nganime-api/config"
	"nganime-api/models"

	"github.com/gin-gonic/gin"
)

type AddCommentInput struct {
	AnimeID  string `json:"anime_id" binding:"required"`
	Content  string `json:"content" binding:"required"`
	ParentID *uint  `json:"parent_id"`
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
		UserID:   userID.(uint),
		AnimeID:  input.AnimeID,
		Content:  input.Content,
		ParentID: input.ParentID,
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
			"parent_id":  comment.ParentID,
			"created_at": comment.CreatedAt,
			"user": gin.H{
				"id":              comment.User.ID,
				"username":        comment.User.Username,
				"profile_picture": comment.User.ProfilePicture,
			},
		},
	})
}

func UpdateComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	commentID := c.Param("id")

	var input struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var comment models.Comment
	if err := config.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	// Check ownership
	if comment.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own comments"})
		return
	}

	comment.Content = input.Content
	config.DB.Save(&comment)

	c.JSON(http.StatusOK, gin.H{"message": "Comment updated successfully", "comment": comment})
}

func DeleteComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	commentID := c.Param("id")

	var comment models.Comment
	if err := config.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	// Check ownership
	if comment.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
		return
	}

	// GORM handles cascade delete via constraint in model
	if err := config.DB.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment and its replies deleted successfully"})
}

func GetCommentsByAnime(c *gin.Context) {
	animeID := c.Param("anime_id")

	var comments []models.Comment
	// Fetch comments and include user details. Order by newest first.
	if err := config.DB.Preload("User").Where("anime_id = ?", animeID).Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}

	// Format response
	var response []gin.H
	for _, comment := range comments {
		response = append(response, gin.H{
			"id":         comment.ID,
			"anime_id":   comment.AnimeID,
			"content":    comment.Content,
			"parent_id":  comment.ParentID,
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
