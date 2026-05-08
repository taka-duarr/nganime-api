package controllers

import (
	"fmt"
	"net/http"
	"nganime-api/config"
	"nganime-api/models"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func UploadProfilePicture(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 1. Get the file from request
	file, err := c.FormFile("profile_picture")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile picture file is required"})
		return
	}

	// 2. Validate file type (basic extension check)
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPG, JPEG, and PNG files are allowed"})
		return
	}

	// 3. Create upload directory if not exists
	uploadDir := "./uploads/profiles"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadDir, 0755)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}
	}

	// 4. Generate unique filename (user_id + timestamp + extension)
	filename := fmt.Sprintf("%d_%d%s", userID.(uint), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// 5. Save the file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// 6. Update user record in database
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	// Store the relative URL
	imageURL := fmt.Sprintf("/uploads/profiles/%s", filename)
	user.ProfilePicture = imageURL

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Profile picture uploaded successfully",
		"profile_picture": imageURL,
	})
}
