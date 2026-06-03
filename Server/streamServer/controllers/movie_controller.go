package controllers

import (
	"context"
	"net/http"
	"stream-server/database"
	"stream-server/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var movieCollection *mongo.Collection = database.OpenCollection("movies")

func GetMovies() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		var movies []models.Movie

		cursor, err := movieCollection.Find(c, bson.M{})

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find movies"})
		}

		defer cursor.Close(c)

		if err = cursor.All(c, &movies); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies"})
		}

		ctx.JSON(http.StatusOK, movies)
	}
}

func GetMovie() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		param, ok := ctx.Params.Get("movieId")

		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		var movie models.Movie
		movieId, err := bson.ObjectIDFromHex(param)

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
			return
		}

		err = movieCollection.FindOne(c, bson.M{"_id": movieId}).Decode(&movie)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
			return
		}

		ctx.JSON(http.StatusOK, movie)
	}
}
