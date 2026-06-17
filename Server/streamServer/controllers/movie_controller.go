package controllers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"stream-server/database"
	"stream-server/models"
	"stream-server/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/openai"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// global variables
var movieCollection *mongo.Collection = database.OpenCollection("movies")
var rankingCollection *mongo.Collection = database.OpenCollection("rankings")
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

type Req struct {
	AdminReview string `json:"admin_review"`
	Rankings    string `json:"rankings"`
}

type Res struct {
	AdminReview string `json:"admin_review"`
	Ranking     models.Ranking
}

func AddReview() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// check for the movide imdbId
		imdbId := ctx.Param("imdb_id")

		if imdbId == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
			return
		}

		var req Req

		// read user review content
		err := ctx.ShouldBind(&req)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad request body"})
			return
		}

		// // get rankings
		rankings, err := GetMovieRankings()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch rankings"})
			return
		}

		var sentimentBuild strings.Builder

		for _, rank := range rankings {
			if rank.RankingValue != 999 {
				sentimentBuild.WriteString(rank.RankingName)
				sentimentBuild.WriteString(", ")
			}
		}

		req.Rankings = strings.Trim(sentimentBuild.String(), ", ")

		// try loading the .env variables
		err = godotenv.Load()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load .env variables"})
			return
		}

		adminReviewSentiment, err := GetAdminReviewSentiment(req)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate admin review sentiment"})
			return
		}

		var rankVal int
		for _, rank := range rankings {
			if adminReviewSentiment == rank.RankingName {
				rankVal = rank.RankingValue
				break
			}
		}

		if rankVal == 0 {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "invalid sentiment generated"})
			return
		}

		filter := bson.M{"imdb_id": imdbId}

		update := bson.M{
			"$set": bson.M{
				"admin_review": req.AdminReview,
				"ranking": bson.M{
					"ranking_value": rankVal,
					"ranking_name":  adminReviewSentiment,
				},
			},
		}

		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		updateRes, err := movieCollection.UpdateOne(c, filter, update)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update movie"})
			return
		}

		if updateRes.ModifiedCount < 1 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "no movie found with that id"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"movie_id": imdbId,
		})
	}
}

func GetMovieRankings() ([]models.Ranking, error) {
	//cleanup context
	c, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()

	// fetching all ranks
	cursor, err := rankingCollection.Find(c, bson.M{})
	if err != nil {
		return nil, err
	}

	defer cursor.Close(c)

	var rankings []models.Ranking

	err = cursor.All(c, &rankings)
	if err != nil {
		return nil, err
	}

	return rankings, nil
}

func GetAdminReviewSentiment(req Req) (string, error) {

	// get the api key
	apiKey := os.Getenv("GROQ_KEY")
	if apiKey == "" {
		return "", errors.New("api key is missing or empty")
	}

	llm, err := openai.New(openai.WithModel("openai/gpt-oss-120b"),
		openai.WithBaseURL("https://api.groq.com/openai/v1"), openai.WithToken(apiKey))

	if err != nil {
		return "", errors.New("failed to load .env variables")
	}

	c, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()

	basePrompt := os.Getenv("BASE_PROMPT")
	if basePrompt == "" {
		return "", errors.New("base prompt is missing or empty")
	}

	basePrompt = strings.Replace(basePrompt, "{ranking}", req.AdminReview, 1)

	res, err := llm.Call(c, basePrompt+req.Rankings)
	if err != nil {
		log.Println("Err: ", err)
		return "", err
	}
	return res, nil
}

func GetRecommendedMovies() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// current user
		userId, err := utils.GetUserIDFromCtx(ctx)

		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// get user favoriteMovies
		favMovies, err := GetUserFavMovies(userId)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if len(favMovies) < 1 {
			return
		}

		movies, err := GetMovieRecommendations(favMovies)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch recommended movies"})
		}

		ctx.JSON(http.StatusOK, gin.H{"movies": movies, "status": "ok"})
	}
}

func GetMovieRecommendations(favGenres []string) ([]models.Movie, error) {
	filters := bson.M{
		"genre.genre_name": bson.M{
			"$in": favGenres,
		},
		"ranking.ranking_value": bson.M{
			"$lt": 4,
		},
	}

	fmt.Println(favGenres)

	opts := options.Find().SetLimit(4)

	// context
	c, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()

	cursor, err := movieCollection.Find(c, filters, opts)
	if err != nil {
		return []models.Movie{}, err
	}

	var movies []models.Movie

	err = cursor.All(c, &movies)

	if err != nil {
		return []models.Movie{}, err
	}
	defer cursor.Close(c)

	return movies, nil
}
