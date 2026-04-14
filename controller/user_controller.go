package controller

import (
	"awesomeProject/environment"
	"awesomeProject/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type UserController struct {
	Gin    *gin.Engine
	db     *mongo.Database
	kc     *kgo.Client
	config *environment.Config
}

func NewUserController(r *gin.Engine, db *mongo.Database, kc *kgo.Client, config *environment.Config) *UserController {
	return &UserController{Gin: r, db: db, kc: kc, config: config}
}

func (uc *UserController) DeleteUser(r *gin.Context) {
	id := r.Param("id")

	userCollection := uc.db.Collection("users")
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		r.JSON(400, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}
	res, err := userCollection.DeleteOne(context.TODO(), bson.M{"_id": objectID})
	if err != nil {
		r.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user", "details": err.Error()})
		return
	}

	//uc.kc.produce(uc.config.KafkaTopic, []byte(id))

	r.JSON(http.StatusOK, gin.H{"deletedCount": res.DeletedCount, "id": objectID})
}

func (uc *UserController) GetUserById(r *gin.Context) {
	id := r.Param("id")

	userCollection := uc.db.Collection("users")
	objectID, err := bson.ObjectIDFromHex(id)

	if err != nil {
		r.JSON(400, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	var user models.User
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		r.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Error()})
		return
	}

	r.JSON(http.StatusOK, gin.H{"user": user})
}

func (uc *UserController) GetUsers(r *gin.Context) {
	userCollection := uc.db.Collection("users")

	cursor, err := userCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		r.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users", "details": err.Error()})
		return
	}

	var users []models.User
	if err = cursor.All(context.TODO(), &users); err != nil {
		r.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users", "details": err.Error()})
		return
	}

	r.JSON(http.StatusOK, gin.H{"users": users})
}

func (uc *UserController) CreateUser(r *gin.Context) {
	var user models.User

	if err := r.BindJSON(&user); err != nil {
		r.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	collection := uc.db.Collection("users")
	result, err := collection.InsertOne(context.TODO(), user)
	if err != nil {
		r.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}
	user.UserId = result.InsertedID.(bson.ObjectID)

	userCreated := models.UserCreated{
		User: user,
		EventDetails: models.EventDetails{
			EventDateTime: time.Now().Format(time.RFC3339),
			EventName:     models.EventTypeUserCreated,
		},
	}
	message, _ := json.Marshal(userCreated)
	err = produceMessage(uc.kc, string(message), uc.config.KafkaTopic)
	if err != nil {
		// Log the error, but respond successfully as the object was created
		fmt.Println("Error producing message: ", err)
	}

	r.JSON(http.StatusCreated, gin.H{"user": user})
}
