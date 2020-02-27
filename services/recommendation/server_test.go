package recommendation

import (
	"context"
	"fmt"
	"log"
	"testing"

	pb "github.com/harlow/go-micro-services/services/recommendation/proto"
)

func TestLoadRecommendations(t *testing.T) {
	mongoClient, err := InitializeDatabase("192.168.31.200:30075")
	if err != nil {
		panic(err)
	}

	srv := Server{
		Tracer:       nil,
		Registry:     nil,
		Port:         0,
		IpAddr:       "",
		MongoClient:  mongoClient,
		RegCheckPort: 0,
	}

	var hotel_map map[string]Hotel
	hotel_map = srv.loadRecommendations()
	log.Printf("load recommendations : %v", hotel_map)
}

func TestGetRecommendations(t *testing.T) {
	mongoClient, err := InitializeDatabase("192.168.31.200:30075")
	if err != nil {
		panic(err)
	}

	srv := Server{
		Tracer:       nil,
		Registry:     nil,
		Port:         0,
		IpAddr:       "",
		MongoClient:  mongoClient,
		RegCheckPort: 0,
	}

	srv.hotels = srv.loadRecommendations()

	ctx, _ := context.WithCancel(context.Background())
	req := pb.Request{Require: "dis", Lat: 37.7867, Lon: -122.4112109150}
	result, err := srv.GetRecommendations(ctx, &req)
	if err != nil {
		fmt.Errorf("%v", err)
	} else {
		fmt.Printf("result : %v", len(result.HotelIds))
	}
}
