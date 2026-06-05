package utils

import (
	"context"
	"errors"
	"os"
	"stream-server/database"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SignedDetails struct {
	UserID    string `bson:"user_id" json:"user_id"`
	FirstName string `bson:"first_name" json:"first_name" validate:"required,min=2,max=100"`
	LastName  string `bson:"last_name" json:"last_name" validate:"required,min=2,max=100"`
	Email     string `bson:"email" json:"email" validate:"required,email"`
	Role      string `bson:"role" json:"role" validate:"oneof=ADMIN USER"`
	jwt.RegisteredClaims
}

// global variables
var TOKEN_SECRET_KEY = os.Getenv("TOKEN_SECRET_KEY")
var REFRESH_SECRET_KEY = os.Getenv("REFRESH_SECRET_KEY")
var userCollection = database.OpenCollection("users")

func GenerateAllTokens(userid, fname, lname, email, role string) (token, refreshToken string, err error) {
	claims := SignedDetails{
		UserID:    userid,
		FirstName: fname,
		LastName:  lname,
		Email:     email,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "streamServer",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	// generate signedToken
	signedToken, err := GenerateToken(claims, TOKEN_SECRET_KEY)
	if err != nil {
		return token, refreshToken, err
	}

	// generate signedRefreshToken
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 7 * 24))
	signedRefreshToken, err := GenerateToken(claims, REFRESH_SECRET_KEY)
	if err != nil {
		return token, refreshToken, err
	}

	return signedToken, signedRefreshToken, err
}

func GenerateToken(claims SignedDetails, secret string) (string, error) {
	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := unsignedToken.SignedString([]byte(secret))

	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func UpdateUserTokens(userId string, token, refreshToken string) error {
	c, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()

	// update the value
	_, err := userCollection.UpdateOne(c, bson.M{"user_id": userId}, bson.M{"$set": bson.M{
		"token":         token,
		"refresh_token": refreshToken,
		"updated_at":    time.Now(),
	}})

	if err != nil {
		return err
	}

	return nil
}

func GetAccessToken(ctx *gin.Context) (string, error) {
	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header is missing")
	}

	token := authHeader[len("Bearer "):]

	if token == "" {
		return "", errors.New("Token is missing")
	}

	return token, nil
}

func ValidateToken(tokenString string) (*SignedDetails, error) {

	var claims = &SignedDetails{}

	// parsing the content to the signedDetails
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return []byte(TOKEN_SECRET_KEY), nil
	})

	if err != nil {
		return nil, err
	}

	// ensuring the right method was used for signing and not an attack
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, err
	}

	// check if token is expired
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}
	return claims, nil
}
