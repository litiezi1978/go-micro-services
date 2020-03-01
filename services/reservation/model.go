package reservation

import (
	"context"
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
	log.Printf("connect to mongo server\n")
	ctx, _ := context.WithCancel(context.Background())
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+url))
	if err != nil {
		log.Fatalf("failed to create mongo client pool, error: %v", err)
	}

	log.Printf("initing room numbers to rate-db\n")
	db := mongoClient.Database("reservation-db")
	collection := db.Collection("reservation")

	cursor, err := collection.Find(ctx, bson.M{"hotelId": "4"})
	if err != nil {
		log.Fatalf("cannot get record from DB, %v", err)
	}
	tempReserves := make([]reservation, 0)
	err = cursor.All(ctx, &tempReserves)
	if err == mongo.ErrNoDocuments || (err == nil && len(tempReserves) == 0) {
		log.Println("insert record to reservation table with hotel id=4")
		newReserv := reservation{"4",
			"Alice",
			"2015-04-09",
			"2015-04-10",
			1}
		_, err = collection.InsertOne(ctx, &newReserv)
		if err != nil {
			log.Fatalf("failed to insert to database, error is: %v", err)
		}
	}

	collection = db.Collection("number")
	for i := 1; i <= 80; i++ {
		hotelId := strconv.Itoa(i)
		cursor, err := collection.Find(ctx, bson.M{"hotelId": hotelId})
		if err != nil {
			log.Fatalf("failed to get record from DB when reading tempNumbers table, error is: %v", err)
		}
		tempNumbers := make([]number, 0)
		err = cursor.All(ctx, &tempNumbers)
		if err == mongo.ErrNoDocuments || (err == nil && len(tempNumbers) == 0) {
			newRoomNumbers := number{}
			newRoomNumbers.HotelId = hotelId
			newRoomNumbers.Number = 200
			if i >= 7 {
				if i%3 == 1 {
					newRoomNumbers.Number = 300
				} else if i%3 == 2 {
					newRoomNumbers.Number = 250
				}
			}
			log.Printf("insert record: %v\n", newRoomNumbers)
			_, err = collection.InsertOne(ctx, &newRoomNumbers)
			if err != nil {
				log.Printf("got error when init db: %v", err)
			}
		}
	}

	return mongoClient, err
}
