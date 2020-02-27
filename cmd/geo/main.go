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
	server_ip := os.Getenv("serverIP")
	server_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	mongoAddr := os.Getenv("mongoAddr")
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")

	log.Printf("geo parameter from env: serverIp=%s, serverPort=%d\n", server_ip, server_port)

	log.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, err := tracing.Init("geo", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, with err=%v", err)
	}

	log.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init Consul, with err=", err)
	}

	log.Printf("init mongo db with addr: %s\n", mongoAddr)
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
