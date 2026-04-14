package main

import (
	"awesomeProject/controller"
	"awesomeProject/models"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	r := gin.Default()

	// Connect to mongoDB
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri).
		SetAuth(options.Credential{
			Username: "admin",
			Password: "password",
		}))
	if err != nil {
		fmt.Println("Error connecting to MongoDB")
		panic(err)
	}
	database := client.Database("users")
	err = models.InitializeDatabase(client, database)
	if err != nil {
		fmt.Println("Error initializing database")
		panic(err)
	}

	userController := controller.NewUserController(r, database)

	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})
	r.GET("/users", userController.GetUsers)
	r.POST("/users", userController.CreateUser)

	r.Run(":8081")
}
