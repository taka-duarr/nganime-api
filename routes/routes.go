package routes

import (
	"nganime-api/controllers"
	"nganime-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Root route
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to NgAnime API",
		})
	})

	api := r.Group("/api")
	
	// Auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/refresh", controllers.RefreshToken)
	}

	// Bookmark routes (Protected)
	bookmarks := api.Group("/bookmarks")
	bookmarks.Use(middleware.AuthMiddleware())
	{
		bookmarks.POST("/", controllers.AddBookmark)
		bookmarks.GET("/", controllers.GetBookmarks)
		bookmarks.DELETE("/:anime_id", controllers.RemoveBookmark)
	}

	// User routes (Protected)
	users := api.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/profile", controllers.GetProfile)
		users.POST("/profile-picture", controllers.UploadProfilePicture)
	}

	// Watch History routes (Protected)
	history := api.Group("/history")
	history.Use(middleware.AuthMiddleware())
	{
		history.POST("/", controllers.MarkAsWatched)
		history.GET("/:anime_id", controllers.GetWatchedEpisodes)
	}

	// Comment routes
	comments := api.Group("/comments")
	{
		// Public: Get comments
		comments.GET("/:anime_id", controllers.GetCommentsByAnime)
		
		// Protected: Add, Update, Delete comment
		comments.POST("/", middleware.AuthMiddleware(), controllers.AddComment)
		comments.PUT("/:id", middleware.AuthMiddleware(), controllers.UpdateComment)
		comments.DELETE("/:id", middleware.AuthMiddleware(), controllers.DeleteComment)
	}

	// Anime API Proxy (Public) — untuk menghindari CORS di Web Browser
	// Semua request ke /api/proxy/* akan diteruskan ke Sanka Vollerei API
	api.Any("/proxy/*path", controllers.ProxyAnimeAPI)
}
