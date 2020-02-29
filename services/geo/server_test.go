package geo

import (
	"context"
	pb "github.com/harlow/go-micro-services/services/geo/proto"
	"github.com/harlow/go-micro-services/tracing"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"testing"
)

func TestServer_Nearby(t *testing.T) {
	srv := Server{
		index:        nil,
		uuid:         "",
		Registry:     nil,
		Tracer:       nil,
		Port:         1234,
		IpAddr:       "localhost",
		RegCheckPort: 12345,
		MongoAddr:    "192.168.31.200:30099",
	}

	tracer, closer, err := tracing.Init("geo-test", "192.168.31.200:30290")
	if err != nil {
		log.Fatalf("failed to init jaeger, with err=%v", err)
	}
	defer closer.Close()
	srv.Tracer = tracer

	ctx, _ := context.WithCancel(context.Background())
	clientOptions := options.Client().ApplyURI("mongodb://" + srv.MongoAddr)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatalf("failed to init mongo %v", err)
	}
	srv.index = NewGeoIndex(mongoClient)

	req := pb.Request{
		Lat: 38.2497,
		Lon: -122.881,
	}

	result, err := srv.Nearby(ctx, &req)
	if err != nil {
		t.Fatalf("failed to call nearby %v", err)
	}

	t.Logf("nearby result %v", result)
}
