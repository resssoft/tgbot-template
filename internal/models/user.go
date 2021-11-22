package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	MongoID        primitive.ObjectID `bson:"_id"`
	TelegramUser   TelegramUser       `bson:"telegramUser"`
	ConversationId string             `bson:"conversationId"`
}
