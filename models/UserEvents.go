package models

type EventDetails struct {
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
