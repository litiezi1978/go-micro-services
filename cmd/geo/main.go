package main

import (
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/geo"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	host_ip := os.Getenv("hostIP")
	server_ip := os.Getenv("serverIP")
	server_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))
	consulPort := os.Getenv("consulPort")
	jaegerPort := os.Getenv("jaegerPort")

	consulAddr := host_ip + ":" + consulPort
	log.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init Consul, with err=", err)
	}

	jaegerAddr := host_ip + ":" + jaegerPort
	log.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("geo", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, with err=%v", err)
	}
	defer closer.Close()

	mongoAddr, err := registry.FindService("srv-mongo-rate")
	if err != nil {
		log.Fatalf("failed to search srv-mongo-rate from Consul, %v", err)
	}
	log.Printf("init mongo DB with addr: %s\n", mongoAddr)

	log.Printf("geo parameter from env: serverIp=%s, serverPort=%d\n", server_ip, server_port)
	srv := &geo.Server{
		Port:         server_port,
		IpAddr:       server_ip,
		Tracer:       tracer,
		Registry:     registry,
		RegCheckPort: consul_check_port,
		MongoAddr:    mongoAddr,
	}
	log.Fatal(srv.Run())
}
