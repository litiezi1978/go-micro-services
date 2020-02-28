package recommendation

import (
	"fmt"
	"log"
	"math"
	"net"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/hailocab/go-geoindex"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/recommendation/proto"
	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/mgo.v2/bson"
)

const name = "srv-recommendation"

type Server struct {
	hotels       map[string]Hotel
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	MongoClient  *mongo.Client
	Registry     *registry.Client
	RegCheckPort int
}

func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	if s.hotels == nil {
		s.hotels = s.loadRecommendations()
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(s.Tracer)),
	)

	pb.RegisterRecommendationServer(srv, s)

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

func (s *Server) GetRecommendations(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	span := opentracing.SpanFromContext(ctx)
	span.LogKV("require", req.Require, "Lat", req.Lat, "Lon", req.Lon)

	res := new(pb.Result)
	log.Printf("GetRecommendations\n")

	require := req.Require
	if require == "dis" {
		p1 := &geoindex.GeoPoint{
			Pid:  "",
			Plat: req.Lat,
			Plon: req.Lon,
		}
		min := math.MaxFloat64
		for _, hotel := range s.hotels {
			tmp := float64(geoindex.Distance(p1, &geoindex.GeoPoint{
				Pid:  "",
				Plat: hotel.HLat,
				Plon: hotel.HLon,
			})) / 1000
			if tmp < min {
				min = tmp
			}
		}
		for _, hotel := range s.hotels {
			tmp := float64(geoindex.Distance(p1, &geoindex.GeoPoint{
				Pid:  "",
				Plat: hotel.HLat,
				Plon: hotel.HLon,
			})) / 1000
			if tmp == min {
				res.HotelIds = append(res.HotelIds, hotel.HId)
			}
		}
	} else if require == "rate" {
		max := 0.0
		for _, hotel := range s.hotels {
			if hotel.HRate > max {
				max = hotel.HRate
			}
		}
		for _, hotel := range s.hotels {
			if hotel.HRate == max {
				res.HotelIds = append(res.HotelIds, hotel.HId)
			}
		}
	} else if require == "price" {
		min := math.MaxFloat64
		for _, hotel := range s.hotels {
			if hotel.HPrice < min {
				min = hotel.HPrice
			}
		}
		for _, hotel := range s.hotels {
			if hotel.HPrice == min {
				res.HotelIds = append(res.HotelIds, hotel.HId)
			}
		}
	} else {
		log.Println("Wrong parameter: ", require)
	}

	return res, nil
}

func (s *Server) loadRecommendations() map[string]Hotel {
	collection := s.MongoClient.Database("recommendation-db").Collection("recommendation")
	ctx, _ := context.WithCancel(context.Background())

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		fmt.Errorf("cannot find any record from recommendation, %v", err)
		return nil
	}

	var hotels []Hotel
	err = cursor.All(ctx, &hotels)
	if err != nil {
		log.Println("Failed get hotels data: ", err)
	}

	profiles := make(map[string]Hotel)
	for _, hotel := range hotels {
		profiles[hotel.HId] = hotel
	}

	return profiles
}
