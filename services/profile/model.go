package profile

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Hotel struct {
	Id          string   `bson:"id"`
	Name        string   `bson:"name"`
	PhoneNumber string   `bson:"phoneNumber"`
	Description string   `bson:"description"`
	Address     *Address `bson:"address"`
}

type Address struct {
	StreetNumber string  `bson:"streetNumber"`
	StreetName   string  `bson:"streetName"`
	City         string  `bson:"city"`
	State        string  `bson:"state"`
	Country      string  `bson:"country"`
	PostalCode   string  `bson:"postalCode"`
	Lat          float32 `bson:"lat"`
	Lon          float32 `bson:"lon"`
}

func InitializeDatabase(url string) (*mongo.Client, error) {
	fmt.Printf("initialize mongodb with url %s\n", url)

	hotels := []Hotel{
		{Id: "1",
			Name:        "Clift Hotel",
			PhoneNumber: "(415) 775-4700",
			Description: "A 6-minute walk from Union Square and 4 minutes from a Muni Metro station, this luxury hotel designed by Philippe Starck features an artsy furniture collection in the lobby, including work by Salvador Dali.",
			Address:     &Address{StreetNumber: "495", StreetName: "Geary St", City: "San Francisco", State: "CA", Country: "United States", PostalCode: "94102", Lat: 37.7867, Lon: -122.4112}},
		{Id: "2",
			Name:        "W San Francisco",
			PhoneNumber: "(415) 777-5300",
			Description: "Less than a block from the Yerba Buena Center for the Arts, this trendy hotel is a 12-minute walk from Union Square.",
			Address:     &Address{StreetNumber: "181", StreetName: "3rd St", City: "San Francisco", State: "CA", Country: "United States", PostalCode: "94103", Lat: 37.7854, Lon: -122.4005}},
		{Id: "3",
			Name:        "Hotel Zetta",
			PhoneNumber: "(415) 543-8555",
			Description: "A 3-minute walk from the Powell Street cable-car turnaround and BART rail station, this hip hotel 9 minutes from Union Square combines high-tech lodging with artsy touches.",
			Address:     &Address{StreetNumber: "55", StreetName: "5th St", City: "San Francisco", State: "CA", Country: "United States", PostalCode: "94103", Lat: 37.7834, Lon: -122.4071}},
		{Id: "4",
			Name:        "Hotel Vitale",
			PhoneNumber: "(415) 278-3700",
			Description: "This waterfront hotel with Bay Bridge views is 3 blocks from the Financial District and a 4-minute walk from the Ferry Building.",
			Address:     &Address{StreetNumber: "8", StreetName: "Mission St", City: "San Francisco", State: "CA", Country: "United States", PostalCode: "94105", Lat: 37.7936, Lon: -122.3930}},
		{Id: "5",
			Name:        "Phoenix Hotel",
			PhoneNumber: "(415) 776-1380",
			Description: "Located in the Tenderloin neighborhood, a 10-minute walk from a BART rail station, this retro motor lodge has hosted many rock musicians and other celebrities since the 1950s. It’s a 4-minute walk from the historic Great American Music Hall nightclub.",
			Address:     &Address{StreetNumber: "601", StreetName: "Eddy St", City: "San Francisco", State: "CA", Country: "United States", PostalCode: "94109", Lat: 37.7831, Lon: -122.4181}},
		{Id: "6",
			Name:        "St. Regis San Francisco",
			PhoneNumber: "(415) 284-4000",
			Description: "St. Regis Museum Tower is a 42-story, 484 ft skyscraper in the South of Market district of San Francisco, California, adjacent to Yerba Buena Gardens, Moscone Center, PacBell Building and the San Francisco Museum of Modern Art.",
			Address:     &Address{StreetNumber: "125", StreetName: "3rd St", City: "San Francisco", State: "CA", Country: "United States", PostalCode: "94109", Lat: 37.7863, Lon: -122.4015}},
	}

	log.Printf("connect to mongo server\n")
	ctx, _ := context.WithCancel(context.Background())
	MongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+url))
	if err != nil {
		fmt.Errorf("cannot connect with mongo, with error = %v", err)
		return nil, err
	}

	log.Printf("get hotels table from mongoDB\n")
	collection := MongoClient.Database("profile-db").Collection("hotels")
	for i := 1; i <= 80; i++ {
		hotel_id := strconv.Itoa(i)

		curr_hotel := Hotel{}
		err = collection.FindOne(ctx, bson.M{"id": hotel_id}).Decode(&curr_hotel)
		if err == mongo.ErrNoDocuments {
			if i < 7 {
				curr_hotel = hotels[i-1]
			} else {
				lat := 37.7835 + float32(i)/500.0*3
				lon := -122.41 + float32(i)/500.0*4

				curr_hotel = Hotel{
					hotel_id,
					"St. Regis San Francisco",
					"(415) 284-40" + hotel_id,
					"St. Regis Museum Tower is a 42-story, 484 ft skyscraper in the South of Market district of San Francisco, California, adjacent to Yerba Buena Gardens, Moscone Center, PacBell Building and the San Francisco Museum of Modern Art.",
					&Address{"125", "3rd St", "San Francisco", "CA", "United States", "94109", lat, lon}}
			}

			_, err = collection.InsertOne(ctx, &curr_hotel)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	return MongoClient, err
}
