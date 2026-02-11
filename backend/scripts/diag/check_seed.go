// scripts/diag/check_seed.go
package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/sohamify/rms-backend/internal/infrastructure/persistence/mongo"
)

func main() {
	godotenv.Load()
	mongo.Init()
	defer mongo.Disconnect()

	ctx := context.Background()
	db := mongo.GetDatabase()

	fmt.Println("=== Roles collection ===")
	cursor, _ := db.Collection("roles").Find(ctx, bson.M{})
	for cursor.Next(ctx) {
		var doc bson.M
		cursor.Decode(&doc)
		fmt.Printf("%+v\n", doc)
	}

	fmt.Println("\n=== Users collection (admin email) ===")
	var user bson.M
	err := db.Collection("users").FindOne(ctx, bson.M{"email": "admin@rms.local"}).Decode(&user)
	if err != nil {
		fmt.Println("No admin user found:", err)
	} else {
		fmt.Printf("%+v\n", user)
	}
}
