package models

const (
	EventTypeUserCreated = "USER_CREATED_EVENT"
	EventTypeUserDeleted = "USER_DELETED_EVENT"
)

type EventDetails struct {
	EventName     string `json:"event_name" required:"true"`
	EventDateTime string `json:"event_at" required:"true"`
}

type UserCreated struct {
	User
	EventDetails
}
type UserUpdated struct {
	User
	EventDetails
}

type UserDeleted struct {
	Id string `json:"id" bson:"_id" required:"true"`
	EventDetails
}
