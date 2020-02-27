package rate

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/rate/proto"
	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/mgo.v2/bson"
)

const name = "srv-rate"

type Server struct {
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	MongoClient  *mongo.Client
	Registry     *registry.Client
	RegCheckPort int
	MemcClient   *memcache.Client
}

func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(s.Tracer),
		),
	)

	fmt.Println("register gRPC server")
	pb.RegisterRateServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = s.Registry.Register(name, s.IpAddr, s.Port, s.RegCheckPort)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}

	return srv.Serve(lis)
}

func (s *Server) Shutdown() {
	s.Registry.Deregister(name)
}

func (s *Server) GetRates(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	res := new(pb.Result)
	ratePlans := make(RatePlans, 0)

	for _, hotelID := range req.HotelIds {
		item, err := s.MemcClient.Get(hotelID)
		if err == nil {
			// memcached hit
			rate_strs := strings.Split(string(item.Value), "\n")
			fmt.Printf("memc hit, hotelId = %s\n", hotelID)
			fmt.Println(rate_strs)

			for _, rate_str := range rate_strs {
				if len(rate_str) != 0 {
					rate_p := pb.RatePlan{}
					json.Unmarshal([]byte(rate_str), &rate_p)
					ratePlans = append(ratePlans, &rate_p)
				}
			}
		} else if err == memcache.ErrCacheMiss {

			fmt.Printf("memc miss, hotelId = %s\n", hotelID)

			// memcached miss, set up mongo connection
			collection := s.MongoClient.Database("rate-db").Collection("inventory")

			memc_str := ""

			tmpRatePlans := make(RatePlans, 0)

			var cursor *mongo.Cursor
			cursor, err = collection.Find(ctx, bson.M{"hotelId": hotelID})
			if err != nil {
				panic(err)
			}
			err = cursor.All(ctx, &tmpRatePlans)
			if err != nil {
				panic(err)
			} else {
				for _, r := range tmpRatePlans {
					ratePlans = append(ratePlans, r)
					rate_json, err := json.Marshal(r)
					if err != nil {
						fmt.Printf("json.Marshal err = %s\n", err)
					}
					memc_str = memc_str + string(rate_json) + "\n"
				}
			}

			// write to memcached
			s.MemcClient.Set(&memcache.Item{Key: hotelID, Value: []byte(memc_str)})

		} else {
			fmt.Printf("Memmcached error = %s\n", err)
			panic(err)
		}
	}

	fmt.Printf("ratePlans data contains: %v", ratePlans)
	sort.Sort(ratePlans)
	res.RatePlans = ratePlans

	return res, nil
}
