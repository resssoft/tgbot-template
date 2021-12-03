package repository

import (
	"github.com/resssoft/tgbot-template/internal/database"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const eventPostsCollectionName = "eventPosts"

type PostRepository interface {
	Add(models.Post) (models.Post, error)
	Remove(string, interface{}) (int64, error)
	Update(primitive.ObjectID, models.Post) error
	GetAll() ([]models.Post, error)
	GetByID(string) (models.Post, error)
	GetByContact(int) (models.Post, error)
	Count() (int64, error)
	GetAllByField(string, interface{}) ([]models.Post, error)
}

type postRepo struct {
	dbApp      database.MongoClientApplication
	collection *mongo.Collection
}

func NewPostRepo(db database.MongoClientApplication) PostRepository {
	collection := db.GetCollection(eventPostsCollectionName)
	return &postRepo{
		dbApp:      db,
		collection: collection,
	}
}

func (u *postRepo) Add(post models.Post) (models.Post, error) {
	post.MongoID = primitive.NewObjectID()
	//log.Debug().Interface("new post", post).Send()
	_, err := u.collection.InsertOne(u.dbApp.GetContext(), post)
	if err != nil {
		log.Error().AnErr("Insert post error", err).Send()
		return post, err
	}
	return post, nil
}

func (u *postRepo) Remove(name string, value interface{}) (int64, error) {
	result, err := u.collection.DeleteMany(u.dbApp.GetContext(), bson.M{name: value})
	if err != nil {
		log.Error().AnErr("Insert post error", err).Send()
		return 0, err
	}

	return result.DeletedCount, nil
}

func (u *postRepo) Update(id primitive.ObjectID, post models.Post) error {
	log.Info().Interface("upd post", post).Send()
	_, err := u.collection.UpdateOne(
		u.dbApp.GetContext(),
		bson.M{"_id": id},
		bson.D{
			{"$set", post},
		})
	if err != nil {
		log.Error().AnErr("Insert post error", err).Send()
		return err
	}
	return nil
}

func (u *postRepo) GetByField(name string, value interface{}) (models.Post, error) {
	post := models.Post{}
	filter := bson.M{name: value}
	//findOptions := new(options.FindOneOptions)
	//findOptions.SetSort(bson.D{{"_id", -1}})
	err := u.collection.FindOne(u.dbApp.GetContext(), filter).Decode(&post)
	if err != nil {
		log.Error().AnErr("post read error", err).Interface(name, value).Send()
		return post, err
	}
	return post, nil
}

func (u *postRepo) GetAllByField(name string, value interface{}) ([]models.Post, error) {
	post := models.Post{}
	filter := bson.M{name: value}
	var posts []models.Post
	cursor, err := u.collection.Find(u.dbApp.GetContext(), filter)
	if err != nil {
		return posts, err
	}
	defer cursor.Close(u.dbApp.GetContext())
	for cursor.Next(u.dbApp.GetContext()) {
		err := cursor.Decode(&post)
		if err != nil {
			log.Error().AnErr("post read error", err).Send()
			continue
		}
		posts = append(posts, post)
	}
	if err := cursor.Err(); err != nil {
		return posts, err
	}
	return posts, nil
}

func (u *postRepo) GetAll() ([]models.Post, error) {
	post := models.Post{}
	var posts []models.Post
	cursor, err := u.collection.Find(u.dbApp.GetContext(), bson.D{})
	if err != nil {
		return posts, err
	}
	defer cursor.Close(u.dbApp.GetContext())
	for cursor.Next(u.dbApp.GetContext()) {
		err := cursor.Decode(&post)
		if err != nil {
			log.Error().AnErr("post read error", err).Send()
			continue
		}
		posts = append(posts, post)
	}
	if err := cursor.Err(); err != nil {
		return posts, err
	}
	return posts, nil
}

func (u *postRepo) GetByID(id string) (models.Post, error) {
	post, err := u.GetByField("id", id)
	return post, err
}

func (u *postRepo) GetByContact(contactId int) (models.Post, error) {
	post, err := u.GetByField("amoCrmData.contactid", contactId)
	return post, err
}

func (u *postRepo) Count() (int64, error) {
	return u.collection.CountDocuments(u.dbApp.GetContext(), bson.D{})
}
