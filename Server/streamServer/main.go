package main

import (
	"log"
	"stream-server/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("health", func(ctx *gin.Context) {
		ctx.String(200, "I am breathing fine")
	})

	// routes and handlers
	routes.UnprotectedRoutes(router)
	routes.ProtectedRoutes(router)

	log.Fatal(router.Run(":8080"))
}
