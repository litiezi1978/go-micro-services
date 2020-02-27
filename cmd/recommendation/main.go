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
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	mongoAddr := os.Getenv("mongoAddr")
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	fmt.Printf("recommendation ip = %s, port = %d\n", serv_ip, serv_port)

	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, err := tracing.Init("recommendation", jaegerAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := recommendation.InitializeDatabase(mongoAddr)
	if err != nil {
		panic(err)
	}

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
