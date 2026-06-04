package main

import (
	"log"
	"stream-server/controllers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("health", func(ctx *gin.Context) {
		ctx.String(200, "I am breathing fine")
	})

	// routes and handlers

	// movie routes
	router.GET("/movies", controllers.GetMovies())
	router.GET("/movies/:imdb_id", controllers.GetMovie())
	router.POST("/movies", controllers.AddMovie())

	// user routes
	router.POST("/register", controllers.RegisterUser())
	router.POST("/login", controllers.LoginUser())
	log.Fatal(router.Run(":8080"))
}
