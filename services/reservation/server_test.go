package reservation

import (
	"context"
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
	pb "github.com/harlow/go-micro-services/services/reservation/proto"
	"github.com/harlow/go-micro-services/tracing"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"testing"
	"time"
)

func TestCheckAvailability(t *testing.T) {
	srv := new(Server)

	jaegerAddr := "192.168.31.200:30290"
	tracer, _, err := tracing.Init("reservation", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, err=%v", err)
	}
	srv.Tracer = tracer

	memcAddr := "192.168.31.200:30063"
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512
	srv.MemcClient = memc_client

	ctx, _ := context.WithCancel(context.Background())
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://192.168.31.200:30065"))
	if err != nil {
		log.Fatalf("failed to create mongo client pool, error: %v", err)
	}
	srv.MongoClient = mongoClient

	req := new(pb.Request)
	req.HotelId = []string{"2", "68", "23", "56", "35"}
	req.InDate = "2018-01-01"
	req.OutDate = "2018-01-03"
	req.RoomNumber = 1
	req.CustomerName = ""

	result, err := srv.CheckAvailability(ctx, req)
	if err != nil {
		log.Fatalf("got error, %v", err)
	}
	fmt.Printf("got result, %v", result)
}
