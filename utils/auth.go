package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
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

// GetSecret mengembalikan JWT secret dari environment.
// Tidak ada nilai bawaan: secret bawaan yang diketahui publik membuat token bisa dipalsukan.
// main.go menolak start jika JWT_SECRET kosong, jadi nilai kosong tidak akan sampai ke sini.
func GetSecret() string {
	return os.Getenv("JWT_SECRET")
}

func getSecret() string {
	return GetSecret()
}

// envInt membaca angka positif dari environment, atau memakai fallback.
func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}

// AccessTokenTTL: umur Access Token. Atur lewat ACCESS_TOKEN_HOURS (bawaan 168 jam = 7 hari).
func AccessTokenTTL() time.Duration {
	return time.Hour * time.Duration(envInt("ACCESS_TOKEN_HOURS", 24*7))
}

// RefreshTokenTTL: umur Refresh Token. Atur lewat REFRESH_TOKEN_DAYS (bawaan 365 hari).
// Refresh Token bersifat rolling: setiap dipakai, diganti dengan yang baru dan umurnya
// dihitung ulang, jadi user yang rutin membuka app tidak akan pernah logout.
func RefreshTokenTTL() time.Duration {
	return time.Hour * 24 * time.Duration(envInt("REFRESH_TOKEN_DAYS", 365))
}

// NewJTI membuat ID acak untuk satu refresh token.
func NewJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateAccessToken membuat Access Token yang terikat ke sebuah sesi (sessionID).
// Middleware memeriksa sesi itu masih aktif, sehingga mencabut sesi langsung
// mematikan access token-nya.
func GenerateAccessToken(userID uint, sessionID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"sid":     sessionID,
		"type":    "access",
		"exp":     time.Now().Add(AccessTokenTTL()).Unix(),
	})
	return token.SignedString([]byte(getSecret()))
}

// GenerateRefreshToken membuat Refresh Token untuk sesi tertentu dengan ID unik (jti).
func GenerateRefreshToken(userID uint, sessionID uint, jti string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"sid":     sessionID,
		"jti":     jti,
		"type":    "refresh",
		"exp":     time.Now().Add(RefreshTokenTTL()).Unix(),
	})
	return token.SignedString([]byte(getSecret()))
}

// RefreshClaims adalah isi Refresh Token yang sudah divalidasi tanda tangan dan umurnya.
type RefreshClaims struct {
	UserID    uint
	SessionID uint   // 0 = token lama (sebelum ada sistem sesi)
	JTI       string // kosong untuk token lama
}

// ParseRefreshToken memvalidasi tanda tangan, umur, dan tipe Refresh Token.
func ParseRefreshToken(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(getSecret()), nil
	})
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token claims")
	}

	// Pastikan ini refresh token, bukan access token
	if tokenType, _ := claims["type"].(string); tokenType != "refresh" {
		return nil, errors.New("token is not a refresh token")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("invalid user_id in token")
	}

	out := &RefreshClaims{UserID: uint(userID)}
	if sid, ok := claims["sid"].(float64); ok {
		out.SessionID = uint(sid)
	}
	out.JTI, _ = claims["jti"].(string)
	return out, nil
}
