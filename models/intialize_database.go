package models

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func InitializeDatabase(mch *mongo.Client, database *mongo.Database) error {
	// 2. Reference your collection
	collection := mch.Database("users").Collection("users")

	// 3. Define the Index Model
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}}, // 1 for Ascending index
		Options: options.Index().
			SetUnique(true).             // Enforce uniqueness
			SetName("unique_email_idx"), // Optional: Name your index
	}

	// 4. Create the index (Safe to run on every service restart)
	name, err := collection.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		// Log error if uniqueness is violated by existing data
		log.Fatalf("Could not create index: %v", err)
		return err
	}

	fmt.Printf("Index initialized successfully: %s\n", name)
	return nil
}
