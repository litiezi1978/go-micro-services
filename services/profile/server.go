package profile

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/profile/proto"
	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/mgo.v2/bson"
)

const name = "srv-profile"

type Server struct {
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	MongoClient  *mongo.Client
	Registry     *registry.Client
	MemcClient   *memcache.Client
	RegCheckPort int
}

func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(s.Tracer)),
	)

	log.Printf("registering Grpc server with name=%s\n", name)
	pb.RegisterProfileServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("registering Consul server\n")
	err = s.Registry.Register(name, s.IpAddr, s.Port, s.RegCheckPort)
	if err != nil {
		log.Fatalf("failed register: %v", err)
	}

	return srv.Serve(lis)
}

func (s *Server) Shutdown() {
	s.Registry.Deregister(name)
}

func (s *Server) GetProfiles(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	span := opentracing.SpanFromContext(ctx)
	span.LogKV("HotelIds", req.HotelIds, "locale", req.Locale)

	res := new(pb.Result)
	hotels := make([]*pb.Hotel, 0)

	for _, i := range req.HotelIds {
		item, err := s.MemcClient.Get(i)
		if err == nil {
			// memcached hit
			log.Printf("memcache hit! with id=%s", i)
			hotel_prof := new(pb.Hotel)
			json.Unmarshal(item.Value, hotel_prof)
			hotels = append(hotels, hotel_prof)

		} else {
			// memcached miss, set up mongo connection
			hotel_prof := new(pb.Hotel)

			collection := s.MongoClient.Database("profile-db").Collection("hotels")
			err = collection.FindOne(ctx, bson.M{"id": i}).Decode(&hotel_prof)
			if err == nil {
				hotels = append(hotels, hotel_prof)

				var prof_json []byte
				prof_json, err = json.Marshal(hotel_prof)
				memc_str := string(prof_json)
				s.MemcClient.Set(&memcache.Item{Key: i, Value: []byte(memc_str)})
			}
		}
	}

	res.Hotels = hotels
	fmt.Printf("In GetProfiles after getting resp\n")
	return res, nil
}
