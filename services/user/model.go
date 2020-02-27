package user

import (
	"context"
	"log"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

func InitializeDatabase(url string) (mongoClient *mongo.Client, err error) {
	ctx, _ := context.WithCancel(context.Background())
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+url))
	if err != nil {
		log.Fatalf("failed to connect mongo %s", err)
	}
	collection := mongoClient.Database("user-db").Collection("user")

	for i := 0; i < 500; i++ {
		suffix := strconv.Itoa(i)
		user_name := "Cornell_" + suffix
		password := ""
		for j := 0; j < 10; j++ {
			password += suffix
		}

		cursor, err := collection.Find(ctx, &bson.M{"username": user_name})
		if err != nil {
			log.Fatalf("failed to get username = %s", user_name)
		}

		my_users := make([]User, 0)
		err = cursor.All(ctx, &my_users)
		if err == mongo.ErrNoDocuments || (err == nil && len(my_users) == 0) {
			_, err = collection.InsertOne(ctx, &User{user_name, password})
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return mongoClient, err
}
