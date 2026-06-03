package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret_key"
	}
	return secret
}

// GenerateAccessToken membuat Access Token yang berlaku 3 hari
func GenerateAccessToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "access",
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // 3 hari
	})
	return token.SignedString([]byte(getSecret()))
}

// GenerateRefreshToken membuat Refresh Token yang berlaku 30 hari
func GenerateRefreshToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(), // 30 hari
	})
	return token.SignedString([]byte(getSecret()))
}

// GenerateJWT adalah alias untuk GenerateAccessToken agar kompatibel dengan kode lama
func GenerateJWT(userID uint) (string, error) {
	return GenerateAccessToken(userID)
}

// ValidateRefreshToken memvalidasi Refresh Token dan mengembalikan user_id
func ValidateRefreshToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(getSecret()), nil
	})

	if err != nil {
		return 0, errors.New("invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid refresh token claims")
	}

	// Pastikan ini adalah refresh token, bukan access token
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return 0, errors.New("token is not a refresh token")
	}

	// Ambil user_id dari claims
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid user_id in token")
	}

	return uint(userIDFloat), nil
}
