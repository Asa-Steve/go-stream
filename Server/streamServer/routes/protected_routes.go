package routes

import (
	"stream-server/controllers"
	"stream-server/middleware"

	"github.com/gin-gonic/gin"
)

func ProtectedRoutes(router *gin.Engine) {
	router.Use(middleware.AuthMiddleware())

	// routes
	router.GET("/movies/:imdb_id", controllers.GetMovie())
	router.POST("/movies", controllers.AddMovie())
}
