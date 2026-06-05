package routes

import (
	"stream-server/controllers"

	"github.com/gin-gonic/gin"
)

func UnprotectedRoutes(router *gin.Engine) {

	// movie routes
	router.GET("/movies", controllers.GetMovies())

	// user routes
	router.POST("/register", controllers.RegisterUser())
	router.POST("/login", controllers.LoginUser())
}
