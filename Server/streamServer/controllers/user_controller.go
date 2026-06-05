package controllers

import (
	"context"
	"net/http"
	"stream-server/database"
	"stream-server/models"
	"stream-server/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashPw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashPw), nil
}

// global variables
var userCollection = database.OpenCollection("users")

func RegisterUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// new user instance
		var user models.User

		// saving / binding the req to the user instance
		err := ctx.ShouldBindJSON(&user)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		// validating
		validate := validator.New()
		err = validate.Struct(&user)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to validate user data"})
			return
		}

		// hashing password
		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		hashedPw, err := HashPassword(user.Password)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error creating user"})
			return
		}

		// checking if user email already exists
		count, err := userCollection.CountDocuments(c, bson.M{"email": user.Email})

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error verifying user email"})
			return
		}

		if count > 0 {
			ctx.JSON(http.StatusConflict, gin.H{"error": "a user with that email exist"})
			return
		}

		// updating the user struct
		user.UserID = bson.NewObjectID().Hex()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		user.Password = hashedPw

		// saving to DB
		_, err = userCollection.InsertOne(c, user)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error creating user"})
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{"message": "user created successfully", "status": "ok"})
	}
}

func LoginUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// new login instance
		var login models.UserLogin

		// binding with req json
		err := ctx.ShouldBindJSON(&login)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		// validate login credentials
		validate := validator.New()
		err = validate.Struct(&login)

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid login credentials"})
			return
		}

		c, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		// user instance to store found user
		var foundUser models.User

		// find user with email
		err = userCollection.FindOne(c, bson.M{"email": login.Email}).Decode(&foundUser)

		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// decrypting
		if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(login.Password)); err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "email or password incorrect"})
			return
		}

		// getting tokens
		token, refreshToken, err := utils.GenerateAllTokens(foundUser.UserID, foundUser.FirstName, foundUser.LastName, foundUser.Email, foundUser.Role)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		// update the user document with the tokens
		err = utils.UpdateUserTokens(foundUser.UserID, token, refreshToken)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update token"})
			return
		}
		
		// building the user response
		var userResponse = models.UserResponse{
			UserID:          foundUser.UserID,
			Email:           foundUser.Email,
			Role:            foundUser.Role,
			FavouriteGenres: foundUser.FavouriteGenres,
			FirstName:       foundUser.FirstName,
			LastName:        foundUser.LastName,
			Token:           token,
			RefreshToken:    refreshToken,
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "login successful", "data": userResponse})
	}
}
