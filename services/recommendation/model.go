package recommendation

import (
	"context"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Hotel struct {
	HId    string  `bson:"hotelId"`
	HLat   float64 `bson:"lat"`
	HLon   float64 `bson:"lon"`
	HRate  float64 `bson:"rate"`
	HPrice float64 `bson:"price"`
}

func InitializeDatabase(url string) (mongoClient *mongo.Client, err error) {
	hotels := []Hotel{
		{"1", 37.7867, -122.4112, 109.00, 150.00},
		{"2", 37.7854, -122.4005, 139.00, 120.00},
		{"3", 37.7834, -122.4071, 109.00, 190.00},
		{"4", 37.7936, -122.3930, 129.00, 160.00},
		{"5", 37.7831, -122.4181, 119.00, 140.00},
		{"6", 37.7863, -122.4015, 149.00, 200.00}}

	fmt.Printf("connect to mongo server\n")
	ctx, _ := context.WithCancel(context.Background())
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+url))
	if err != nil {
		panic(err)
	}

	fmt.Printf("create recommendation table for recommendation-db\n")
	collection := mongoClient.Database("recommendation-db").Collection("recommendation")
	for i := 1; i <= 80; i++ {
		hotel_id := strconv.Itoa(i)
		//fmt.Printf("find record for hotel with id=%s\n", hotel_id)

		curr_hotel := Hotel{}
		err = collection.FindOne(ctx, bson.M{"hotelId": hotel_id}).Decode(&curr_hotel)
		if err == mongo.ErrNoDocuments {
			if i < 7 {
				curr_hotel = hotels[i-1]
			} else {
				lat := 37.7835 + float64(i)/500.0*3
				lon := -122.41 + float64(i)/500.0*4
				rate := 135.00
				rate_inc := 179.00
				if i%3 == 0 {
					if i%5 == 0 {
						rate = 109.00
						rate_inc = 123.17
					} else if i%5 == 1 {
						rate = 120.00
						rate_inc = 140.00
					} else if i%5 == 2 {
						rate = 124.00
						rate_inc = 144.00
					} else if i%5 == 3 {
						rate = 132.00
						rate_inc = 158.00
					} else if i%5 == 4 {
						rate = 232.00
						rate_inc = 258.00
					}
				}

				curr_hotel = Hotel{hotel_id, lat, lon, rate, rate_inc}
			}

			_, err = collection.InsertOne(ctx, &curr_hotel)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return mongoClient, err
}
