package database

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect() *mongo.Client {
	err := godotenv.Load(".env")

	if err != nil {
		log.Println("warning: env file not loaded")
	}

	MongoDB := os.Getenv("MONGODB_URI")
	if MongoDB == "" {
		log.Fatal("Err: MONGODB_URI not set")
	}

	clientOptions := options.Client().ApplyURI(MongoDB)
	client, err := mongo.Connect(clientOptions)

	if err != nil {
		return nil
	}

	return client
}

var Client = Connect()

func OpenCollection(collectionName string) *mongo.Collection {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("warning: env file not loaded")
	}

	DBName := os.Getenv("DATABASE_NAME")

	if DBName == "" {
		log.Fatal("Err: DATABASE_NAME not set")
	}

	collection := Client.Database(DBName).Collection(collectionName)

	if collection == nil {
		return nil
	}

	return collection
}
