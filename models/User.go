package models

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	UserId bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name   string        `bson:"name" json:"name"`
	Age    int           `bson:"age" json:"age"`
	Email  string        `bson:"email" json:"email"`
}

func NewUser() *User {
	return &User{}
}
