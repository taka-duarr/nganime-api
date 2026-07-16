# NgAnime API

NgAnime API adalah backend (REST API) yang dibangun menggunakan [Golang](https://golang.org/) dan framework [Gin Gonic](https://gin-gonic.com/). Backend ini dirancang untuk mendampingi aplikasi NgAnime (frontend React Native / Expo) dengan menyediakan fitur-fitur seperti autentikasi, manajemen pengguna, riwayat tontonan, sistem bookmark, serta *video streaming proxy*.

## 🚀 Fitur Utama

- **User Authentication:** Registrasi dan Login menggunakan sistem JWT (JSON Web Token) dengan skema *Access Token* dan *Refresh Token*.
- **User Profile:** Manajemen profil pengguna, termasuk fitur upload dan ubah foto profil (*avatar*).
- **Watch History:** Menyimpan dan mengambil riwayat tontonan episode (terakhir ditonton) untuk setiap anime.
- **Bookmarks:** Sistem favorit/bookmark agar pengguna bisa menyimpan daftar anime yang ingin mereka tonton.
- **Video Proxy / Unpacker:** Sistem *reverse-proxy* dan ekstraktor (seperti HLS / m3u8 proxy) untuk mem-bypass masalah CORS saat melakukan *streaming* video dari server CDN pihak ketiga.

## 🛠️ Persyaratan Sistem

- [Go](https://golang.org/dl/) versi 1.19 atau lebih baru.
- [MySQL](https://www.mysql.com/) atau MariaDB (via XAMPP/Laragon dll).
- Git.

## 📦 Instalasi & Menjalankan Lokal

1. **Clone Repository (Jika belum)**
   ```bash
   git clone <repo-url>
   cd nganime-api
   ```

2. **Install Dependensi**
   Download semua package yang dibutuhkan (Gin, GORM, JWT, dsb):
   ```bash
   go mod tidy
   ```

3. **Konfigurasi Environment (`.env`)**
   Buat file bernama `.env` di root direktori (atau sesuaikan file yang sudah ada), lalu isi konfigurasi databasenya:
   ```ini
   DB_USER=root
   DB_PASSWORD=
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_NAME=nganime_db

   JWT_SECRET=supersecretkey_nganime2026
   PORT=8080

   # URL Anime API upstream (digunakan oleh Proxy Controller)
   ANIME_API_BASE_URL=https://www.sankavollerei.web.id/anime
   ```
   *(Pastikan database `nganime_db` sudah terbuat di MySQL kamu sebelum menjalankan aplikasi).*

4. **Menjalankan Server (Development)**
   Gunakan perintah berikut untuk menjalankan server secara lokal (mendukung hot-reload jika menggunakan *tools* seperti `air`, tapi secara bawaan gunakan `go run`):
   ```bash
   go run main.go
   ```
   Aplikasi akan berjalan di `http://localhost:8080`.

5. **Build untuk Production (Deployment)**
   Untuk kompilasi menjadi file binary yang siap di-deploy (misalnya ke VPS Linux / Windows):
   ```bash
   # Build untuk sistem operasi saat ini (Windows)
   go build -o nganime-api.exe main.go

   # Build untuk Linux (jika kamu ingin mendeploy ke VPS Linux Ubuntu/CentOS)
   GOOS=linux GOARCH=amd64 go build -o nganime-api-linux main.go
   ```

## 📂 Struktur Direktori Utama

- `config/` - Konfigurasi koneksi ke Database.
- `controllers/` - Logika *routing* dari masing-masing endpoint (Auth, Bookmark, History, Video Proxy).
- `middleware/` - Middleware proteksi otorisasi JWT untuk rute yang di-protect.
- `models/` - Definisi skema tabel GORM (User, WatchHistory, Bookmark).
- `routes/` - Tempat semua URL *endpoints* API di daftarkan.
- `utils/` - Kumpulan fungsi *helper*, seperti *hashing* password, pembuatan JWT, dan manipulasi token.
- `uploads/` - Folder tempat foto profil pengguna disimpan.

## 🔗 Panduan Endpoint Singkat

### Auth API
- `POST /api/register` - Daftar akun baru.
- `POST /api/login` - Login untuk mendapatkan token JWT.
- `POST /api/refresh` - Refresh token akses (menggunakan *refresh token*).

### Protected API (Wajib Header `Authorization: Bearer <token>`)
- `GET /api/me` - Mendapatkan info user saat ini.
- `POST /api/profile-picture` - Upload foto profil pengguna.
- `POST /api/history` - Menyimpan history tontonan.
- `GET /api/history/:anime_id` - Mengambil history anime tertentu.
- `GET /api/bookmarks` - Melihat daftar anime yang di-bookmark.
- `POST /api/bookmarks` - Menyimpan anime ke bookmark.
- `DELETE /api/bookmarks/:anime_id` - Menghapus anime dari bookmark.

### Proxy / Public API
- `GET /api/video-proxy` - Streaming proxy endpoint untuk mengatasi CORS HLS CDN pihak ketiga.

---

**Dibuat oleh Tim NgAnime**
