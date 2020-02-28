package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/user"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))

	mongoAddr := os.Getenv("mongoAddr")
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	fmt.Printf("user ip = %s, port = %d\n", serv_ip, serv_port)

	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("hotel-user", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, err=%v", err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init consul, err=%v", err)
	}

	fmt.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := user.InitializeDatabase(mongoAddr)
	if err != nil {
		log.Fatalf("failed to init mongo, err=%v", err)
	}

	srv := &user.Server{
		Tracer:       tracer,
		Registry:     registry,
		Port:         serv_port,
		IpAddr:       serv_ip,
		MongoClient:  mongoClient,
		RegCheckPort: consul_check_port,
	}
	log.Fatal(srv.Run())
}
