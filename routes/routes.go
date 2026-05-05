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
	}

	// Bookmark routes (Protected)
	bookmarks := api.Group("/bookmarks")
	bookmarks.Use(middleware.AuthMiddleware())
	{
		bookmarks.POST("/", controllers.AddBookmark)
		bookmarks.GET("/", controllers.GetBookmarks)
		bookmarks.DELETE("/:anime_id", controllers.RemoveBookmark)
	}
}
