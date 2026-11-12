package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nganime-api/config"
	"nganime-api/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type authEnv struct {
	t *testing.T
	r *gin.Engine
}

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Session{}, &models.Bookmark{}, &models.Comment{}, &models.WatchHistory{}); err != nil {
		t.Fatal(err)
	}
	config.DB = db
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r)
	return &authEnv{t: t, r: r}
}

func (e *authEnv) do(method, path, bearer string, body interface{}) (int, map[string]interface{}) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	out := map[string]interface{}{}
	json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func (e *authEnv) login(username string) (access, refresh string) {
	e.t.Helper()
	code, out := e.do("POST", "/api/auth/login", "", map[string]string{"username": username, "password": "secret123"})
	if code != http.StatusOK {
		e.t.Fatalf("login gagal: %d %v", code, out)
	}
	return out["token"].(string), out["refresh_token"].(string)
}

func (e *authEnv) register(username string) {
	e.t.Helper()
	if code, out := e.do("POST", "/api/auth/register", "", map[string]string{"username": username, "password": "secret123"}); code != http.StatusCreated {
		e.t.Fatalf("register gagal: %d %v", code, out)
	}
}

func TestLoginRefreshLogoutFlow(t *testing.T) {
	e := newAuthEnv(t)
	e.register("budi")
	access, refresh := e.login("budi")

	if code, _ := e.do("GET", "/api/users/profile", access, nil); code != http.StatusOK {
		t.Fatalf("access token hasil login harus bisa dipakai, dapat %d", code)
	}
	// refresh token tidak boleh dipakai sebagai access token
	if code, _ := e.do("GET", "/api/users/profile", refresh, nil); code != http.StatusUnauthorized {
		t.Fatalf("refresh token tidak boleh mengakses API, dapat %d", code)
	}

	// refresh: dapat pasangan baru, token baru bekerja
	code, out := e.do("POST", "/api/auth/refresh", "", map[string]string{"refresh_token": refresh})
	if code != http.StatusOK {
		t.Fatalf("refresh harus berhasil, dapat %d %v", code, out)
	}
	access2, refresh2 := out["token"].(string), out["refresh_token"].(string)
	if refresh2 == refresh {
		t.Fatal("refresh token harus dirotasi")
	}
	if code, _ := e.do("GET", "/api/users/profile", access2, nil); code != http.StatusOK {
		t.Fatalf("access token baru harus bekerja, dapat %d", code)
	}

	// logout mencabut sesi: access token dan refresh token sama-sama mati
	if code, _ := e.do("POST", "/api/auth/logout", "", map[string]string{"refresh_token": refresh2}); code != http.StatusOK {
		t.Fatalf("logout harus 200, dapat %d", code)
	}
	if code, _ := e.do("GET", "/api/users/profile", access2, nil); code != http.StatusUnauthorized {
		t.Fatalf("access token setelah logout harus ditolak, dapat %d", code)
	}
	if code, _ := e.do("POST", "/api/auth/refresh", "", map[string]string{"refresh_token": refresh2}); code != http.StatusUnauthorized {
		t.Fatalf("refresh setelah logout harus ditolak, dapat %d", code)
	}
}

func TestLogoutAllRevokesEveryDevice(t *testing.T) {
	e := newAuthEnv(t)
	e.register("sari")
	accessA, refreshA := e.login("sari") // perangkat A
	accessB, refreshB := e.login("sari") // perangkat B

	if code, _ := e.do("POST", "/api/auth/logout-all", accessA, nil); code != http.StatusOK {
		t.Fatalf("logout-all harus 200, dapat %d", code)
	}
	for name, tok := range map[string]string{"A": accessA, "B": accessB} {
		if code, _ := e.do("GET", "/api/users/profile", tok, nil); code != http.StatusUnauthorized {
			t.Fatalf("access token perangkat %s harus ditolak, dapat %d", name, code)
		}
	}
	for name, tok := range map[string]string{"A": refreshA, "B": refreshB} {
		if code, _ := e.do("POST", "/api/auth/refresh", "", map[string]string{"refresh_token": tok}); code != http.StatusUnauthorized {
			t.Fatalf("refresh perangkat %s harus ditolak, dapat %d", name, code)
		}
	}
}

func TestStolenRefreshTokenReuseRevokesSession(t *testing.T) {
	e := newAuthEnv(t)
	e.register("andi")
	_, stolen := e.login("andi")

	// pemilik sah merotasi token
	code, out := e.do("POST", "/api/auth/refresh", "", map[string]string{"refresh_token": stolen})
	if code != http.StatusOK {
		t.Fatalf("refresh pertama harus berhasil, dapat %d", code)
	}
	legit := out["refresh_token"].(string)
	legitAccess := out["token"].(string)

	// lewat masa tenggang, pencuri memakai token lama
	config.DB.Model(&models.Session{}).Where("1 = 1").Update("rotated_at", time.Now().Add(-10*time.Minute))
	if code, _ := e.do("POST", "/api/auth/refresh", "", map[string]string{"refresh_token": stolen}); code != http.StatusUnauthorized {
		t.Fatalf("pemakaian ulang token lama harus ditolak, dapat %d", code)
	}
	// seluruh sesi mati, termasuk milik pemilik sah
	if code, _ := e.do("GET", "/api/users/profile", legitAccess, nil); code != http.StatusUnauthorized {
		t.Fatalf("access token sesi yang dicabut harus ditolak, dapat %d", code)
	}
	if code, _ := e.do("POST", "/api/auth/refresh", "", map[string]string{"refresh_token": legit}); code != http.StatusUnauthorized {
		t.Fatalf("refresh sesi yang dicabut harus ditolak, dapat %d", code)
	}
}

func TestLogoutWithGarbageTokenStill200(t *testing.T) {
	e := newAuthEnv(t)
	if code, _ := e.do("POST", "/api/auth/logout", "", map[string]string{"refresh_token": "bukan-token"}); code != http.StatusOK {
		t.Fatalf("logout dengan token rusak tetap harus 200, dapat %d", code)
	}
	if code, _ := e.do("POST", "/api/auth/logout", "", nil); code != http.StatusOK {
		t.Fatalf("logout tanpa body tetap harus 200, dapat %d", code)
	}
}
