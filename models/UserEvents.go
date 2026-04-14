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
	User         `json:",inline" bson:",inline"`
	EventDetails `json:",inline" bson:",inline"`
}
type UserUpdated struct {
	User         `json:",inline" bson:",inline"`
	EventDetails `json:",inline" bson:",inline"`
}

type UserDeleted struct {
	Id           string `json:"id" bson:"_id" required:"true"`
	EventDetails `json:",inline" bson:",inline"`
}
