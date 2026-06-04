package controllers

import (
	"context"
	"fmt"
	"net/http"
	"stream-server/database"
	"stream-server/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// global variables
var movieCollection *mongo.Collection = database.OpenCollection("movies")
var validate = validator.New()

// Get all movies
func GetMovies() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		var movies []models.Movie

		cursor, err := movieCollection.Find(c, bson.M{})

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find movies"})
			return
		}

		defer cursor.Close(c)

		if err = cursor.All(c, &movies); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies"})
			return
		}

		ctx.JSON(http.StatusOK, movies)
	}
}

// Get a single movie
func GetMovie() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		imdb_id := ctx.Param("imdb_id")

		if imdb_id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "movie id is required"})
			return
		}

		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		var movie models.Movie
		fmt.Println(imdb_id)

		err := movieCollection.FindOne(c, bson.M{"imdb_id": imdb_id}).Decode(&movie)

		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}

		ctx.JSON(http.StatusOK, movie)
	}
}

// Add a movie
func AddMovie() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// creating a context with timeout for cleanup
		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		// creating a movie instance
		var movie models.Movie

		err := ctx.ShouldBindJSON(&movie)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request", "error": err.Error()})
			return
		}

		// validating the movie struct
		err = validate.Struct(movie)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid movie data", "error": err.Error()})
			return
		}

		// adding to the collection
		result, err := movieCollection.InsertOne(c, movie)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to add movie", "error": err.Error()})
			return
		}

		ctx.JSON(http.StatusCreated, result)
	}
}

