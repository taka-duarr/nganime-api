package controllers

import (
	"errors"
	"net/http"
	"nganime-api/config"
	"nganime-api/models"
	"nganime-api/services"
	"nganime-api/utils"

	"github.com/gin-gonic/gin"
)

type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := config.DB.Where("username = ?", input.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username: input.Username,
		Password: hashedPassword,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Registration successful", "user": gin.H{"id": user.ID, "username": user.Username}})
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Setiap login = satu sesi baru (bisa dicabut kapan saja)
	session, err := services.CreateSession(config.DB, user.ID, c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	accessToken, refreshToken, err := issueTokens(user.ID, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Login successful",
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user":          gin.H{"id": user.ID, "username": user.Username, "profile_picture": user.ProfilePicture},
	})
}

// RefreshToken menerima refresh_token dan mengembalikan access_token + refresh_token baru (rolling)
func RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	// Validasi tanda tangan, umur, dan tipe refresh token
	claims, err := utils.ParseRefreshToken(body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token. Please login again."})
		return
	}

	// Pastikan user masih ada di database
	var user models.User
	if err := config.DB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found. Please login again."})
		return
	}

	userAgent := c.GetHeader("User-Agent")

	var session *models.Session
	if claims.SessionID == 0 {
		// Token lama (diterbitkan sebelum sistem sesi ada): tukar sekali menjadi sesi baru.
		// Token lama berumur maksimal 30 hari, jadi jalur ini hilang dengan sendirinya.
		session, err = services.CreateSession(config.DB, user.ID, userAgent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
			return
		}
	} else {
		// Rotasi: hanya refresh token terbaru yang sah; pemakaian ulang token lama mencabut sesi.
		session, err = services.RotateSession(config.DB, claims, userAgent)
		if err != nil {
			if errors.Is(err, services.ErrSessionInvalid) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session revoked or expired. Please login again."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh session"})
			return
		}
	}

	newAccessToken, newRefreshToken, err := issueTokens(user.ID, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         newAccessToken,
		"refresh_token": newRefreshToken,
	})
}

// issueTokens menerbitkan access token dan refresh token untuk sebuah sesi.
func issueTokens(userID uint, session *models.Session) (accessToken, refreshToken string, err error) {
	accessToken, err = utils.GenerateAccessToken(userID, session.ID)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = utils.GenerateRefreshToken(userID, session.ID, session.CurrentJTI)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// Logout mencabut sesi milik refresh token yang dikirim (logout di perangkat ini).
// Selalu membalas 200 supaya app bisa logout lokal walau token sudah kedaluwarsa.
func Logout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err == nil && body.RefreshToken != "" {
		if claims, err := utils.ParseRefreshToken(body.RefreshToken); err == nil && claims.SessionID > 0 {
			if err := services.RevokeSession(config.DB, claims.SessionID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session"})
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// LogoutAll mencabut semua sesi milik user yang sedang login (semua perangkat).
func LogoutAll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if err := services.RevokeAllSessions(config.DB, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke sessions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices"})
}
