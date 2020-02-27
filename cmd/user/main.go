package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/user"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	host_ip := os.Getenv("hostIP")
	serv_ip := os.Getenv("serverIP")

	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	mongo_port := os.Getenv("mongoPort")
	jaeger_port := os.Getenv("jaegerPort")
	consul_port := os.Getenv("consulPort")
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	fmt.Printf("user ip = %s, port = %d\n", serv_ip, serv_port)

	jaegerAddr := fmt.Sprintf("%s:%s", host_ip, jaeger_port)
	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)

	tracer, err := tracing.Init("user", jaegerAddr)
	if err != nil {
		panic(err)
	}

	consulAddr := fmt.Sprintf("%s:%s", host_ip, consul_port)
	fmt.Printf("init consul with addr: %s\n", consulAddr)

	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	mongoAddr := fmt.Sprintf("%s:%s", host_ip, mongo_port)
	fmt.Printf("init mongo DB with addr: %s\n", mongoAddr)

	mongoClient, err := user.InitializeDatabase(mongoAddr)
	if err != nil {
		panic(err)
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
