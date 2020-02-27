package profile

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	pb "github.com/harlow/go-micro-services/services/profile/proto"
)

func TestGetProfiles(t *testing.T) {

	mongoClient, err := InitializeDatabase("192.168.31.200:30098")
	if err != nil {
		panic(err)
	}

	memc_client := memcache.New("192.168.31.200:30096")
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
		HotelIds: []string{"1", "2"},
		Locale:   "zh_cn",
	}

	var result *pb.Result
	result, err = srv.GetProfiles(ctx, &req)
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Printf("%v", result)
}
