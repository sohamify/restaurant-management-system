package mongo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	once     sync.Once
	client   *mongo.Client
	database *mongo.Database
)

func Init() error {
	once.Do(func() {
		uri := os.Getenv("MONGO_URI")
		if uri == "" {
			panic("MONGO_URI environment variable is not set")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to MongoDB: %v", err))
		}

		// Verify connection
		if err = client.Ping(ctx, nil); err != nil {
			panic(fmt.Sprintf("MongoDB ping failed: %v", err))
		}

		dbName := os.Getenv("MONGO_DB_NAME")
		if dbName == "" {
			dbName = "rms" // default database name
		}

		database = client.Database(dbName)
		fmt.Printf("Successfully connected to MongoDB database: %s\n", database.Name())
	})

	return nil
}

func GetClient() *mongo.Client {
	if client == nil {
		panic("MongoDB client not initialized. Call mongo.Init() first.")
	}
	return client
}

func GetDatabase() *mongo.Database {
	if database == nil {
		panic("MongoDB database not initialized. Call mongo.Init() first.")
	}
	return database
}

func Disconnect() error {
	if client != nil {
		return client.Disconnect(context.Background())
	}
	return nil
}
