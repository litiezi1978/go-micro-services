package reservation

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type reservation struct {
	HotelId      string `bson:"hotelId"`
	CustomerName string `bson:"customerName"`
	InDate       string `bson:"inDate"`
	OutDate      string `bson:"outDate"`
	Number       int    `bson:"number"`
}

type number struct {
	HotelId string `bson:"hotelId"`
	Number  int    `bson:"numberOfRoom"`
}

func InitializeDatabase(url string) (mongoClient *mongo.Client, err error) {
	fmt.Printf("connect to mongo server\n")
	ctx, _ := context.WithCancel(context.Background())
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+url))
	if err != nil {
		panic(err)
	}

	fmt.Printf("read inventory table from rate-db\n")
	db := mongoClient.Database("reservation-db")
	collection := db.Collection("reservation")

	reservations := make([]reservation, 0)

	cursor, err := collection.Find(ctx, bson.M{"hotelId": "4"})
	fmt.Errorf("my err is %v", err)
	if err != nil {
		panic(err)
	}
	err = cursor.All(ctx, &reservations)
	fmt.Printf("reservations is %v\n", reservations)
	if err == mongo.ErrNoDocuments || (err == nil && len(reservations) == 0) {
		fmt.Println("insert record to reservation table with hotel id=4")
		reserv := reservation{"4", "Alice", "2015-04-09", "2015-04-10", 1}
		_, err = collection.InsertOne(ctx, &reserv)
		if err != nil {
			log.Fatal(err)
		}
	}

	collection = db.Collection("number")

	for i := 1; i <= 80; i++ {
		hotelId := strconv.Itoa(i)

		var numbers []number
		cursor, err := collection.Find(ctx, bson.M{"hotelId": hotelId})
		if err != nil {
			log.Fatal(err)
		}
		err = cursor.All(ctx, &numbers)
		if err == mongo.ErrNoDocuments || (err == nil && len(numbers) == 0) {
			room_num := 200
			if i >= 7 {
				if i%3 == 1 {
					room_num = 300
				} else if i%3 == 2 {
					room_num = 250
				}

			}
			fmt.Printf("insert number record with number with %d\n", room_num)
			_, err = collection.InsertOne(ctx, &number{hotelId, room_num})
			if err != nil {
				fmt.Errorf("got error when init db: %v", err)
			}
		}
	}

	return mongoClient, err
}
