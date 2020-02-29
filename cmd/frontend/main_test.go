package main

import (
	"context"
	"fmt"
	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/frontend"
	reservepb "github.com/harlow/go-micro-services/services/reservation/proto"
	"github.com/harlow/go-micro-services/tracing"
	"github.com/opentracing/opentracing-go"
	"log"
	"testing"
)

func TestMain(m *testing.M) {
	jaegerAddr := "192.168.30.200:30290"
	tracer, closer, err := tracing.Init("frontend", jaegerAddr)
	if err != nil {
		log.Fatalf("Cannot init Jaeger tracer, err=%v", err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	consulAddr := "192.168.30.200:30085"
	log.Printf("init consul with %s", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	s := frontend.Server{}
	s.IpAddr = "127.0.0.1"
	s.Port = 8087
	s.Tracer = tracer
	s.Registry = registry

	if err := s.InitSearchClient("srv-search"); err != nil {
		fmt.Errorf("%v", err)
		return
	}
	if err := s.InitProfileClient("srv-profile"); err != nil {
		fmt.Errorf("%v", err)
		return
	}
	if err := s.InitRecommendationClient("srv-recommendation"); err != nil {
		fmt.Errorf("%v", err)
		return
	}
	if err := s.InitUserClient("srv-user"); err != nil {
		fmt.Errorf("%v", err)
		return
	}
	if err := s.InitReservation("srv-reservation"); err != nil {
		fmt.Errorf("%v", err)
		return
	}

	ctx, _ := context.WithCancel(context.Background())

	reservReq := reservepb.Request{
		CustomerName: "",
		HotelId:      []string{"1", "2", "3"},
		InDate:       "2005-01-01",
		OutDate:      "2005-03-01",
		RoomNumber:   1,
	}
	reservationResp, err := s.ReservationClient.CheckAvailability(ctx, &reservReq)
	fmt.Printf("%v", reservationResp)
}
