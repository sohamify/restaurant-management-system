// scripts/seed/seed_initial_data.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/sohamify/rms-backend/internal/infrastructure/persistence/mongo" // adjust module name
)

func main() {
	if err := godotenv.Load(); err != nil { // adjust path if needed
		fmt.Println("No .env file found – using system env")
	}

	if err := mongo.Init(); err != nil {
		log.Fatal("Mongo init failed:", err)
	}
	defer mongo.Disconnect()

	ctx := context.Background()
	db := mongo.GetDatabase()

	// ── Create ADMIN role ───────────────────────────────────────
	rolesColl := db.Collection("roles")

	adminRole := bson.M{
		"name": "ADMIN",
		"permissions": []string{
			"USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_VIEW_ALL",
			"ROLE_MANAGE", "MENU_MANAGE", "TABLE_MANAGE", "REPORT_VIEW_ALL",
			"ORDER_VIEW_ALL", "PAYMENT_OVERRIDE", "INVENTORY_MANAGE", "SETTINGS_UPDATE",
			// add more from your RBAC design
		},
		"created_at": time.Now(),
	}

	// Upsert to avoid duplicates
	res, err := rolesColl.UpdateOne(ctx,
		bson.M{"name": "ADMIN"},
		bson.M{"$setOnInsert": adminRole},
	)
	if err != nil {
		log.Fatal("Failed to upsert ADMIN role:", err)
	}
	var adminRoleID primitive.ObjectID
	if res.MatchedCount == 0 {
		// newly inserted → fetch the _id
		var insertedRole struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		rolesColl.FindOne(ctx, bson.M{"name": "ADMIN"}).Decode(&insertedRole)
		adminRoleID = insertedRole.ID
	} else {
		// already existed → fetch existing id
		var existing struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		rolesColl.FindOne(ctx, bson.M{"name": "ADMIN"}).Decode(&existing)
		adminRoleID = existing.ID
	}

	// ── Create admin user ───────────────────────────────────────
	usersColl := db.Collection("users")

	password := "ChangeMe123!" // ← CHANGE THIS IN REALITY
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Password hash failed:", err)
	}

	adminUser := bson.M{
		"first_name":    "System",
		"last_name":     "Administrator",
		"email":         "admin@rms.local",
		"password_hash": string(hash),
		"role_id":       adminRoleID,
		"is_active":     true,
		"phone":         "9876543210",
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
		"deleted_at":    nil,
	}

	_, err = usersColl.UpdateOne(ctx,
		bson.M{"email": "admin@rms.local"},
		bson.M{"$setOnInsert": adminUser},
	)
	if err != nil {
		log.Fatal("Failed to upsert admin user:", err)
	}

	fmt.Println("=======================================")
	fmt.Println("  Initial data seeded successfully!")
	fmt.Println("  Admin email:    admin@rms.local")
	fmt.Println("  Admin password: ChangeMe123!")
	fmt.Println("  !!! CHANGE PASSWORD IMMEDIATELY !!!")
	fmt.Println("=======================================")
}
