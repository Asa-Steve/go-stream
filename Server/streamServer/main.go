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

	router.GET("/movies", controllers.GetMovies())
	router.GET("/movies/:movieId", controllers.GetMovie())

	log.Fatal(router.Run(":8080"))
}
