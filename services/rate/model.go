package rate

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"strconv"
	"time"

	pb "github.com/harlow/go-micro-services/services/rate/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RatePlans []*pb.RatePlan

func (r RatePlans) Len() int {
	return len(r)
}

func (r RatePlans) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RatePlans) Less(i, j int) bool {
	return r[i].RoomType.TotalRate > r[j].RoomType.TotalRate
}

type RoomType struct {
	BookableRate       float64 `bson:"bookableRate"`
	Code               string  `bson:"code"`
	RoomDescription    string  `bson:"roomDescription"`
	TotalRate          float64 `bson:"totalRate"`
	TotalRateInclusive float64 `bson:"totalRateInclusive"`
}

type RatePlan struct {
	HotelId  string    `bson:"hotelId"`
	Code     string    `bson:"code"`
	InDate   string    `bson:"inDate"`
	OutDate  string    `bson:"outDate"`
	RoomType *RoomType `bson:"roomType"`
}

func InitializeDatabase(url string) (mongoClient *mongo.Client, err error) {
	ratePlans := [3]RatePlan{
		{"1",
			"RACK",
			"2015-04-09",
			"2015-04-10",
			&RoomType{
				109.00,
				"KNG",
				"King sized bed",
				109.00,
				123.17}},
		{"2",
			"RACK",
			"2015-04-09",
			"2015-04-10",
			&RoomType{
				139.00,
				"QN",
				"Queen sized bed",
				139.00,
				153.09}},
		{"3",
			"RACK",
			"2015-04-09",
			"2015-04-10",
			&RoomType{
				109.00,
				"KNG",
				"King sized bed",
				109.00,
				123.17}},
	}

	log.Printf("connecting to mongo server...\n")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+url))
	if err != nil {
		log.Fatalf("failed to connect to mongo, err=%v\n", err)
	}
	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatalf("failed to connect to mongo err=%v\n", err)
	}

	log.Printf("reading inventory table from rate-db...\n")
	collection := mongoClient.Database("rate-db").Collection("inventory")

	for i := 1; i <= 80; i++ {
		hotel_id := strconv.Itoa(i)

		mongoRatePlans := make([]RatePlan, 0)
		cursor, err := collection.Find(ctx, bson.M{"hotelId": hotel_id})
		if err != nil {
			log.Fatalf("failed to read mongodb, err=%v", err)
		}
		err = cursor.All(ctx, &mongoRatePlans)
		if (err == mongo.ErrNoDocuments) || len(mongoRatePlans) == 0 {
			curr_ratePlan := RatePlan{}
			if i <= 3 {
				curr_ratePlan = ratePlans[i-1]
			} else {
				end_date := "2015-04-"
				rate := 109.00
				rate_inc := 123.17
				if i%2 == 0 {
					end_date = end_date + "17"
				} else {
					end_date = end_date + "24"
				}
				if i%5 == 1 {
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
				curr_ratePlan = RatePlan{hotel_id,
					"RACK",
					"2015-04-09",
					end_date,
					&RoomType{
						rate,
						"KNG",
						"King sized bed",
						rate,
						rate_inc},
				}
			}

			_, err = collection.InsertOne(ctx, &curr_ratePlan)
			if err != nil {
				fmt.Errorf("insert docu when error occur, %v", err)
			}
		}
	}

	return mongoClient, err
}
