package controller

import (
	"awesomeProject/models"
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type UserController struct {
	Gin *gin.Engine
	db  *mongo.Database
}

func NewUserController(r *gin.Engine, db *mongo.Database) *UserController {
	return &UserController{Gin: r, db: db}
}

func (uc *UserController) GetUser(r *gin.Context) {

}

func (uc *UserController) GetUsers(r *gin.Context) {
	userCollection := uc.db.Collection("users")

	cursor, err := userCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		r.JSON(500, gin.H{"error": "Failed to fetch users", "details": err.Error()})
		return
	}

	var users []models.User
	if err = cursor.All(context.TODO(), &users); err != nil {
		r.JSON(500, gin.H{"error": "Failed to fetch users", "details": err.Error()})
		return
	}

	r.JSON(200, gin.H{"users": users})
}

func (uc *UserController) CreateUser(r *gin.Context) {
	var user models.User

	if err := r.BindJSON(&user); err != nil {
		r.JSON(400, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	collection := uc.db.Collection("users")
	result, err := collection.InsertOne(context.TODO(), user)
	if err != nil {
		r.JSON(500, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}
	user.UserId = result.InsertedID.(bson.ObjectID)

	r.JSON(201, gin.H{"user": user})
}
