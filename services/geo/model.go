package geo

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type point struct {
	Pid  string  `bson:"hotelId"`
	Plat float64 `bson:"lat"`
	Plon float64 `bson:"lon"`
}

// Implement Point interface
func (p *point) Lat() float64 {
	return p.Plat
}

func (p *point) Lon() float64 {
	return p.Plon
}

func (p *point) Id() string {
	return p.Pid
}

func initializeDatabase(url string) (*mongo.Client, error) {
	points := [6]point{
		{Pid: "1", Plat: 37.7867, Plon: -122.4112},
		{Pid: "2", Plat: 37.7854, Plon: -122.4005},
		{Pid: "3", Plat: 37.7854, Plon: -122.4071},
		{Pid: "4", Plat: 37.7936, Plon: -122.3930},
		{Pid: "5", Plat: 37.7831, Plon: -122.4181},
		{Pid: "6", Plat: 37.7863, Plon: -122.4015}}

	ctx, _ := context.WithCancel(context.Background())
	clientOptions := options.Client().ApplyURI("mongodb://" + url)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("initializing database, and failed to connect to mongo, %v/n", err)
	}

	collection := mongoClient.Database("geo-db").Collection("geo")

	for i := 1; i <= 80; i++ {
		hotel_id := strconv.Itoa(i)

		curr_point := point{}
		err = collection.FindOne(ctx, bson.M{"hotelId": hotel_id}).Decode(&curr_point)
		if err != nil && err == mongo.ErrNoDocuments {
			if i < 7 {
				curr_point = points[i-1]
			} else {
				lat := 37.7835 + float64(i)/500.0*3.7
				lat, _ = strconv.ParseFloat(fmt.Sprintf("%.4f", lat), 64)
				lon := -122.4102 + float64(i)/500.0*4.2
				lon, _ = strconv.ParseFloat(fmt.Sprintf("%.4f", lon), 64)

				curr_point = point{
					Pid:  hotel_id,
					Plat: lat,
					Plon: lon,
				}
			}

			_, err = collection.InsertOne(ctx, &curr_point)
			if err != nil {
				log.Printf("insert point record when error: %v/n", err)
			}
		}
	}

	return mongoClient, err
}

func NewGeoIndex(mongoClient *mongo.Client) []point {
	points := make([]point, 80)

	ctx, _ := context.WithCancel(context.Background())
	collection := mongoClient.Database("geo-db").Collection("geo")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatalf("failed to get point record from mongo, with err=%v", err)
	}
	err = cursor.All(ctx, &points)
	if err != nil {
		log.Fatalf("failed to read cursor from mongo, with err=%v", err)
	}
	log.Printf("newGeoIndex points = %v\n", points)
	return points
}
