package controller

import (
	"awesomeProject/environment"
	"awesomeProject/models"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var testDB *mongo.Database
var testClient *mongo.Client
var testConfig = environment.Config{
	MongoURI:   "mongodb://localhost:27017",
	KafkaTopic: "test-topic",
}

type MockKafkaProducer struct {
	mu       sync.Mutex
	Messages []*kgo.Record
}

func (m *MockKafkaProducer) Produce(ctx context.Context, r *kgo.Record, cb func(*kgo.Record, error)) {
	m.mu.Lock()
	m.Messages = append(m.Messages, r)
	m.mu.Unlock()
	go cb(r, nil)
}

func (m *MockKafkaProducer) Clear() {
	m.mu.Lock()
	m.Messages = nil
	m.mu.Unlock()
}

var mockKP = &MockKafkaProducer{}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Connect to local MongoDB for testing
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
		// If we can't connect, skip tests that need DB or fail early
		// For this task, we assume a local DB is available as it's common in such environments
		panic(err)
	}
	testClient = client
	testDB = client.Database("test_db")

	// Run tests
	code := m.Run()

	// Cleanup
	_ = testDB.Drop(context.TODO())
	_ = client.Disconnect(context.TODO())

	os.Exit(code)
}

func setupRouter() (*gin.Engine, *UserController) {
	r := gin.New()
	uc := NewUserController(r, testDB, mockKP, &testConfig)
	r.POST("/users", uc.CreateUser)
	r.GET("/users", uc.GetUsers)
	r.GET("/users/:id", uc.GetUserById)
	r.DELETE("/users/:id", uc.DeleteUser)
	return r, uc
}

func TestCreateUser(t *testing.T) {
	mockKP.Clear()
	r, _ := setupRouter()

	user := models.User{
		Name:  "Test User",
		Age:   25,
		Email: "test@example.com",
	}
	jsonUser, _ := json.Marshal(user)

	req, _ := http.NewRequestWithContext(t.Context(), "POST", "/users", bytes.NewBuffer(jsonUser))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]models.User
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	createdUser := response["user"]
	if createdUser.Name != user.Name || createdUser.Email != user.Email {
		t.Errorf("Response user mismatch. Expected %v, got %v", user, createdUser)
	}
	if createdUser.UserId.IsZero() {
		t.Error("Expected UserId to be set")
	}

	// Verify Kafka message
	if len(mockKP.Messages) != 1 {
		t.Errorf("Expected 1 Kafka message, got %d", len(mockKP.Messages))
	} else {
		msg := mockKP.Messages[0]
		if msg.Topic != testConfig.KafkaTopic {
			t.Errorf("Expected topic %s, got %s", testConfig.KafkaTopic, msg.Topic)
		}
		var userCreated models.UserCreated
		if err := json.Unmarshal(msg.Value, &userCreated); err != nil {
			t.Errorf("Failed to unmarshal Kafka message: %v", err)
		}
		if userCreated.Name != user.Name || userCreated.Email != user.Email {
			t.Errorf("Kafka message content mismatch. Expected %v, got %v", user, userCreated.User)
		}
		if userCreated.EventName != models.EventTypeUserCreated {
			t.Errorf("Expected event name %s, got %s", models.EventTypeUserCreated, userCreated.EventName)
		}
		// Verify flat structure (JSON unmarshal to map should have fields at top level)
		var flatMap map[string]any
		_ = json.Unmarshal(msg.Value, &flatMap)
		if _, ok := flatMap["name"]; !ok {
			t.Error("Expected 'name' field at top level of Kafka message")
		}
		if _, ok := flatMap["event_name"]; !ok {
			t.Error("Expected 'event_name' field at top level of Kafka message")
		}
	}
}

func TestGetUsers(t *testing.T) {
	r, _ := setupRouter()
	_ = testDB.Collection("users").Drop(t.Context())

	// Seed data
	testUsers := []models.User{
		{Name: "User 1", Age: 20, Email: "u1@e.com"},
		{Name: "User 2", Age: 30, Email: "u2@e.com"},
	}
	_, _ = testDB.Collection("users").InsertMany(t.Context(), testUsers)

	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string][]models.User
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	users := response["users"]
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestGetUserById(t *testing.T) {
	r, _ := setupRouter()

	// Seed data
	user := models.User{Name: "Single User", Age: 40, Email: "single@e.com"}
	res, err := testDB.Collection("users").InsertOne(t.Context(), user)
	if err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}
	id := res.InsertedID.(bson.ObjectID).Hex()

	// Test successful case
	req, _ := http.NewRequestWithContext(t.Context(), "GET", "/users/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]models.User
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if response["user"].Name != user.Name {
		t.Errorf("Expected user name %s, got %s", user.Name, response["user"].Name)
	}

	// Test not found case
	fakeId := bson.NewObjectID().Hex()
	req, _ = http.NewRequestWithContext(t.Context(), "GET", "/users/"+fakeId, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for fake ID, got %d", w.Code)
	}

	// Test invalid ID case
	req, _ = http.NewRequestWithContext(t.Context(), "GET", "/users/invalid-id", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	mockKP.Clear()
	r, _ := setupRouter()

	// Seed data
	user := models.User{Name: "To Delete", Age: 50, Email: "delete@e.com"}
	res, err := testDB.Collection("users").InsertOne(t.Context(), user)
	if err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}
	id := res.InsertedID.(bson.ObjectID).Hex()

	// Test successful case
	req, _ := http.NewRequestWithContext(t.Context(), "DELETE", "/users/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if response["deletedCount"].(float64) != 1 {
		t.Errorf("Expected deletedCount 1, got %v", response["deletedCount"])
	}

	// Verify Kafka message
	if len(mockKP.Messages) != 1 {
		t.Errorf("Expected 1 Kafka message, got %d", len(mockKP.Messages))
	} else {
		msg := mockKP.Messages[0]
		var userDeleted models.UserDeleted
		if err := json.Unmarshal(msg.Value, &userDeleted); err != nil {
			t.Errorf("Failed to unmarshal Kafka message: %v", err)
		}
		if userDeleted.Id != id {
			t.Errorf("Expected deleted ID %s, got %s", id, userDeleted.Id)
		}
		if userDeleted.EventName != models.EventTypeUserDeleted {
			t.Errorf("Expected event name %s, got %s", models.EventTypeUserDeleted, userDeleted.EventName)
		}
	}

	// Test not found case
	mockKP.Clear()
	fakeId := bson.NewObjectID().Hex()
	req, _ = http.NewRequestWithContext(t.Context(), "DELETE", "/users/"+fakeId, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for fake ID, got %d", w.Code)
	}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	if response["deletedCount"].(float64) != 0 {
		t.Errorf("Expected deletedCount 0 for fake ID, got %v", response["deletedCount"])
	}
	// Even if not found in DB, current implementation still produces message?
	// Let's check:
	// res, err := userCollection.DeleteOne(...)
	// if err != nil { ... return }
	// userDeleted := ...
	// produceMessage(...)
	// Yes, it currently produces message even if deletedCount is 0.
	if len(mockKP.Messages) != 1 {
		t.Errorf("Expected 1 Kafka message for fake ID, got %d", len(mockKP.Messages))
	}

	// Test invalid ID case
	mockKP.Clear()
	req, _ = http.NewRequestWithContext(t.Context(), "DELETE", "/users/invalid-id", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
	if len(mockKP.Messages) != 0 {
		t.Errorf("Expected 0 Kafka messages for invalid ID, got %d", len(mockKP.Messages))
	}
}
