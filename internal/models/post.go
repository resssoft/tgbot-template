package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type PostButtonType string

const LinkButton PostButtonType = "link"
const SignUpButton PostButtonType = "sign-up"
const SignOutButton PostButtonType = "sign-out"

type Post struct {
	MongoID     primitive.ObjectID `bson:"_id"`
	Name        string
	Buttons     []PostButton
	AdminId     int64
	AdminLogin  string
	Description string
	Image       string
}

type PostButton struct {
	Type PostButtonType
	Text string
	Url  string
	Data string
}
