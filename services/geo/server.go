package geo

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/geo/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	name             = "srv-geo"
	maxSearchRadius  = 10
	maxSearchResults = 5
)

type Server struct {
	index        []point
	uuid         string
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	RegCheckPort int
	MongoAddr    string
}

func (s *Server) Run() error {
	ctx, err := initializeDatabase(s.MongoAddr)
	if err != nil {
		log.Fatalf("init database failed: %v", err)
	}

	if s.Port == 0 {
		log.Fatalf("server port must be set")
	}

	if s.index == nil {
		s.index = NewGeoIndex(ctx)
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(s.Tracer)),
	)

	pb.RegisterGeoServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = s.Registry.Register(name, s.IpAddr, s.Port, s.RegCheckPort)
	if err != nil {
		log.Fatalf("failed register: %v", err)
	}

	return srv.Serve(lis)
}

func (s *Server) Shutdown() {
	s.Registry.Deregister(name)
}

func (s *Server) Nearby(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	log.Printf("In geo Nearby\n")
	span := opentracing.SpanFromContext(ctx)
	span.LogKV("Lon", req.Lon, "Lat", req.Lat)
	lat, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", req.Lat), 64)
	lon, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", req.Lon), 64)

	points := s.getNearbyPoints(ctx, lat, lon)
	log.Printf("geo after getNearbyPoints, len = %d\n", len(points))

	res := &pb.Result{}
	for _, p := range points {
		log.Printf("In geo Nearby return hotelId = %s\n", p.Id())
		res.HotelIds = append(res.HotelIds, p.Id())
	}

	span.LogKV("RespHotelIds", res.HotelIds)
	return res, nil
}

func (s *Server) getNearbyPoints(ctx context.Context, lat, lon float64) []point {
	log.Printf("In geo getNearbyPoints, lat = %f, lon = %f\n", lat, lon)
	result := make([]point, 0)
	for i := 0; i < 5; i++ {
		idx := rand.Intn(79)
		result = append(result, s.index[idx])
	}
	return result
}
