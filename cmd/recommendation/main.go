package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/recommendation"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	host_ip := os.Getenv("hostIP")
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))
	consulPort := os.Getenv("consulPort")
	jaegerPort := os.Getenv("jaegerPort")

	consulAddr := host_ip + ":" + consulPort
	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init consul, err=%v", err)
	}

	jaegerAddr := host_ip + ":" + jaegerPort
	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("recommendation", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, err=%v", err)
	}
	defer closer.Close()

	mongoAddr, err := registry.FindService("srv-mongo-recomm")
	if err != nil {
		log.Fatalf("failed to search srv-mongo-recomm from Consul, %v", err)
	}
	log.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := recommendation.InitializeDatabase(mongoAddr)
	if err != nil {
		log.Fatalf("failed to init mongo, err=%v", err)
	}

	fmt.Printf("recommendation ip = %s, port = %d\n", serv_ip, serv_port)
	srv := &recommendation.Server{
		Tracer:       tracer,
		Registry:     registry,
		Port:         serv_port,
		IpAddr:       serv_ip,
		MongoClient:  mongoClient,
		RegCheckPort: consul_check_port,
	}
	log.Fatal(srv.Run())
}
