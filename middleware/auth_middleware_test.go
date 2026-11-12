package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nganime-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func call(r *gin.Engine, token string) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w.Code
}

// stubSession mengganti pemeriksaan sesi ke database dengan fungsi tes.
func stubSession(t *testing.T, active func(sid uint) bool) {
	t.Helper()
	old := IsSessionActive
	IsSessionActive = active
	t.Cleanup(func() { IsSessionActive = old })
}

func TestAccessTokenAccepted(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	stubSession(t, func(uint) bool { return true })
	tok, _ := utils.GenerateAccessToken(1, 7)
	if got := call(newRouter(), tok); got != http.StatusOK {
		t.Fatalf("access token harus diterima, dapat %d", got)
	}
}

func TestRevokedSessionRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	stubSession(t, func(uint) bool { return false })
	tok, _ := utils.GenerateAccessToken(1, 7)
	if got := call(newRouter(), tok); got != http.StatusUnauthorized {
		t.Fatalf("access token dari sesi yang dicabut harus ditolak, dapat %d", got)
	}
}

func TestSessionIDPassedToChecker(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	var seen uint
	stubSession(t, func(sid uint) bool { seen = sid; return true })
	tok, _ := utils.GenerateAccessToken(1, 42)
	call(newRouter(), tok)
	if seen != 42 {
		t.Fatalf("middleware harus memeriksa sesi 42, dapat %d", seen)
	}
}

func TestLegacyTokenWithoutSessionAccepted(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	stubSession(t, func(uint) bool { t.Fatal("token lama tanpa sid tidak boleh cek sesi"); return false })
	legacy, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("test-secret-0123456789"))
	if got := call(newRouter(), legacy); got != http.StatusOK {
		t.Fatalf("token lama (tanpa sid) harus tetap diterima sampai kedaluwarsa, dapat %d", got)
	}
}

func TestRefreshTokenRejectedAsAccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	stubSession(t, func(uint) bool { return true })
	tok, _ := utils.GenerateRefreshToken(1, 7, "jti-abc")
	if got := call(newRouter(), tok); got != http.StatusUnauthorized {
		t.Fatalf("refresh token tidak boleh dipakai sebagai access token, dapat %d", got)
	}
}

func TestWrongSecretRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-lama-0123456789")
	stubSession(t, func(uint) bool { return true })
	tok, _ := utils.GenerateAccessToken(1, 7)
	t.Setenv("JWT_SECRET", "secret-baru-0123456789")
	if got := call(newRouter(), tok); got != http.StatusUnauthorized {
		t.Fatalf("token dengan secret berbeda harus ditolak, dapat %d", got)
	}
}

func TestMissingHeaderRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	if got := call(newRouter(), ""); got != http.StatusUnauthorized {
		t.Fatalf("tanpa token harus ditolak, dapat %d", got)
	}
}

func TestTokenLifetimes(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_HOURS", "")
	t.Setenv("REFRESH_TOKEN_DAYS", "")
	if utils.AccessTokenTTL().Hours() != 24*7 {
		t.Fatalf("access TTL bawaan harus 7 hari, dapat %v", utils.AccessTokenTTL())
	}
	if utils.RefreshTokenTTL().Hours() != 24*365 {
		t.Fatalf("refresh TTL bawaan harus 365 hari, dapat %v", utils.RefreshTokenTTL())
	}
	t.Setenv("REFRESH_TOKEN_DAYS", "90")
	if utils.RefreshTokenTTL().Hours() != 24*90 {
		t.Fatalf("REFRESH_TOKEN_DAYS harus bisa diatur, dapat %v", utils.RefreshTokenTTL())
	}
}
