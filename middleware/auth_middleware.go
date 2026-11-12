package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"nganime-api/config"
	"nganime-api/services"
	"nganime-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// IsSessionActive memeriksa sesi ke database. Dibuat sebagai variabel agar bisa
// diganti di tes tanpa database.
var IsSessionActive = func(sessionID uint) bool {
	return services.IsSessionActive(config.DB, sessionID)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret := utils.GetSecret()

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Refresh Token tidak boleh dipakai untuk mengakses API (hanya untuk /auth/refresh)
		if tokenType, _ := claims["type"].(string); tokenType == "refresh" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Access token terikat ke sesi: sesi yang sudah dicabut (logout / logout semua
		// perangkat / pencurian token terdeteksi) langsung menolak access token-nya.
		// Token lama tanpa "sid" (diterbitkan sebelum sistem sesi) tetap diterima sampai
		// kedaluwarsa dan akan diganti token bersesi saat refresh.
		if sid, hasSID := claims["sid"].(float64); hasSID && sid > 0 {
			if !IsSessionActive(uint(sid)) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session revoked or expired"})
				c.Abort()
				return
			}
			c.Set("session_id", uint(sid))
		}

		c.Set("user_id", uint(userID))
		c.Next()
	}
}
