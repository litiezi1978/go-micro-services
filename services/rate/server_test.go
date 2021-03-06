package rate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	pb "github.com/harlow/go-micro-services/services/rate/proto"
)

func TestGetRates(t *testing.T) {

	mongoClient, err := InitializeDatabase("192.168.31.200:30095")
	if err != nil {
		panic(err)
	}

	memc_client := memcache.New("192.168.31.200:30093")
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	srv := Server{
		Tracer:       nil,
		Registry:     nil,
		Port:         0,
		IpAddr:       "",
		MongoClient:  mongoClient,
		MemcClient:   memc_client,
		RegCheckPort: 0,
	}

	ctx, _ := context.WithCancel(context.Background())

	req := pb.Request{
		HotelIds: []string{"68", "71", "69", "28", "42"},
		InDate:   "2015-01-01",
		OutDate:  "2016-03-01",
	}

	var result *pb.Result
	result, err = srv.GetRates(ctx, &req)
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Printf("%v", result)
}
