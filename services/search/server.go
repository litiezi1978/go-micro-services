package search

import (
	// "encoding/json"
	"fmt"
	otlog "github.com/opentracing/opentracing-go/log"

	// F"io/ioutil"
	"log"
	"net"

	// "os"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/dialer"
	"github.com/harlow/go-micro-services/registry"
	geo "github.com/harlow/go-micro-services/services/geo/proto"
	rate "github.com/harlow/go-micro-services/services/rate/proto"
	pb "github.com/harlow/go-micro-services/services/search/proto"
	opentracing "github.com/opentracing/opentracing-go"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const name = "srv-search"

type Server struct {
	geoClient  geo.GeoClient
	rateClient rate.RateClient

	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	Registry     *registry.Client
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
	pb.RegisterSearchServer(srv, s)

	// init grpc clients
	if err := s.initGeoClient("srv-geo"); err != nil {
		return err
	}
	if err := s.initRateClient("srv-rate"); err != nil {
		return err
	}

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

func (s *Server) initGeoClient(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.geoClient = geo.NewGeoClient(conn)
	return nil
}

func (s *Server) initRateClient(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.rateClient = rate.NewRateClient(conn)
	return nil
}

func (s *Server) Nearby(ctx context.Context, req *pb.NearbyRequest) (*pb.SearchResult, error) {
	log.Printf("received request req=%v", req)
	span := opentracing.SpanFromContext(ctx)
	span.LogKV("Nearby_req", *req)

	nearby, err := s.geoClient.Nearby(ctx, &geo.Request{
		Lat: req.Lat,
		Lon: req.Lon,
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))
		log.Fatalf("nearby error: %v", err)
	}
	span.LogKV("Nearby_resp", nearby.HotelIds)
	log.Printf("get nearby json from geo service: %v", nearby)

	rateReq := rate.Request{
		HotelIds: nearby.HotelIds,
		InDate:   req.InDate,
		OutDate:  req.OutDate,
	}
	span.LogKV("rateReq", rateReq)
	log.Printf("send req to rate req=%v", rateReq)
	rates, err := s.rateClient.GetRates(ctx, &rateReq)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))
		log.Fatalf("rates error: %v", err)
	}
	span.LogKV("rateResp", rates)
	log.Printf("get rate json from rates service: %v", rates)

	// TODO(hw): add simple ranking algo to order hotel ids:
	// * geo distance
	// * price (best discount?)
	// * reviews

	// build the response
	res := pb.SearchResult{}
	for _, ratePlan := range rates.RatePlans {
		fmt.Printf("get RatePlan HotelId = %s, Code = %s\n", ratePlan.HotelId, ratePlan.Code)
		res.HotelIds = append(res.HotelIds, ratePlan.HotelId)
	}
	return &res, nil
}
