package services

import (
	"testing"
	"time"

	"nganime-api/models"
	"nganime-api/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-0123456789")
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Session{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func claimsFor(s *models.Session, jti string) *utils.RefreshClaims {
	return &utils.RefreshClaims{UserID: s.UserID, SessionID: s.ID, JTI: jti}
}

func TestCreateAndRotate(t *testing.T) {
	db := testDB(t)
	s, err := CreateSession(db, 1, "ua")
	if err != nil {
		t.Fatal(err)
	}
	if !IsSessionActive(db, s.ID) {
		t.Fatal("sesi baru harus aktif")
	}

	first := s.CurrentJTI
	rotated, err := RotateSession(db, claimsFor(s, first), "ua")
	if err != nil {
		t.Fatalf("rotasi dengan token terbaru harus berhasil: %v", err)
	}
	if rotated.CurrentJTI == first {
		t.Fatal("JTI harus berganti setelah rotasi")
	}
	// token baru bisa dirotasi lagi
	if _, err := RotateSession(db, claimsFor(s, rotated.CurrentJTI), "ua"); err != nil {
		t.Fatalf("token baru harus bisa dirotasi: %v", err)
	}
}

func TestGraceWindowAllowsRacingRefresh(t *testing.T) {
	db := testDB(t)
	s, _ := CreateSession(db, 1, "ua")
	old := s.CurrentJTI
	if _, err := RotateSession(db, claimsFor(s, old), "ua"); err != nil {
		t.Fatal(err)
	}
	// token lama dipakai lagi sesaat setelah dirotasi (mis. tab kedua): masih diterima
	if _, err := RotateSession(db, claimsFor(s, old), "ua"); err != nil {
		t.Fatalf("token lama dalam masa tenggang harus diterima: %v", err)
	}
	if !IsSessionActive(db, s.ID) {
		t.Fatal("sesi tidak boleh dicabut karena race refresh")
	}
}

func TestReuseOutsideGraceRevokesSession(t *testing.T) {
	db := testDB(t)
	s, _ := CreateSession(db, 1, "ua")
	old := s.CurrentJTI
	current, err := RotateSession(db, claimsFor(s, old), "ua")
	if err != nil {
		t.Fatal(err)
	}
	// simulasikan masa tenggang sudah lewat
	past := time.Now().Add(-5 * time.Minute)
	db.Model(&models.Session{}).Where("id = ?", s.ID).Update("rotated_at", past)

	if _, err := RotateSession(db, claimsFor(s, old), "ua"); err != ErrSessionInvalid {
		t.Fatalf("pemakaian ulang token lama harus ditolak, dapat %v", err)
	}
	if IsSessionActive(db, s.ID) {
		t.Fatal("sesi harus dicabut saat pemakaian ulang terdeteksi")
	}
	// pemilik sah pun harus login ulang: token terbaru ikut mati
	if _, err := RotateSession(db, claimsFor(s, current.CurrentJTI), "ua"); err != ErrSessionInvalid {
		t.Fatalf("token terbaru dari sesi yang dicabut harus ditolak, dapat %v", err)
	}
}

func TestRevokeSession(t *testing.T) {
	db := testDB(t)
	s, _ := CreateSession(db, 1, "ua")
	other, _ := CreateSession(db, 1, "ua")
	if err := RevokeSession(db, s.ID); err != nil {
		t.Fatal(err)
	}
	if IsSessionActive(db, s.ID) {
		t.Fatal("sesi yang dicabut tidak boleh aktif")
	}
	if !IsSessionActive(db, other.ID) {
		t.Fatal("sesi lain tidak boleh ikut dicabut")
	}
	if _, err := RotateSession(db, claimsFor(s, s.CurrentJTI), "ua"); err != ErrSessionInvalid {
		t.Fatalf("refresh pada sesi yang dicabut harus ditolak, dapat %v", err)
	}
}

func TestRevokeAllSessionsOnlyAffectsThatUser(t *testing.T) {
	db := testDB(t)
	a1, _ := CreateSession(db, 1, "ua")
	a2, _ := CreateSession(db, 1, "ua")
	b, _ := CreateSession(db, 2, "ua")
	if err := RevokeAllSessions(db, 1); err != nil {
		t.Fatal(err)
	}
	if IsSessionActive(db, a1.ID) || IsSessionActive(db, a2.ID) {
		t.Fatal("semua sesi user 1 harus dicabut")
	}
	if !IsSessionActive(db, b.ID) {
		t.Fatal("sesi user lain tidak boleh terpengaruh")
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	db := testDB(t)
	s, _ := CreateSession(db, 1, "ua")
	db.Model(&models.Session{}).Where("id = ?", s.ID).Update("expires_at", time.Now().Add(-time.Hour))
	if IsSessionActive(db, s.ID) {
		t.Fatal("sesi kedaluwarsa tidak boleh aktif")
	}
	if _, err := RotateSession(db, claimsFor(s, s.CurrentJTI), "ua"); err != ErrSessionInvalid {
		t.Fatalf("sesi kedaluwarsa harus ditolak, dapat %v", err)
	}
}

func TestWrongUserRejected(t *testing.T) {
	db := testDB(t)
	s, _ := CreateSession(db, 1, "ua")
	c := claimsFor(s, s.CurrentJTI)
	c.UserID = 99
	if _, err := RotateSession(db, c, "ua"); err != ErrSessionInvalid {
		t.Fatalf("user_id tidak cocok harus ditolak, dapat %v", err)
	}
}

func TestCleanupRemovesOldOnly(t *testing.T) {
	db := testDB(t)
	active, _ := CreateSession(db, 1, "ua")
	expired, _ := CreateSession(db, 1, "ua")
	db.Model(&models.Session{}).Where("id = ?", expired.ID).Update("expires_at", time.Now().Add(-time.Hour))
	CleanupSessions(db)

	var n int64
	db.Model(&models.Session{}).Where("id = ?", active.ID).Count(&n)
	if n != 1 {
		t.Fatal("sesi aktif tidak boleh dihapus")
	}
	db.Model(&models.Session{}).Where("id = ?", expired.ID).Count(&n)
	if n != 0 {
		t.Fatal("sesi kedaluwarsa harus dihapus")
	}
}
