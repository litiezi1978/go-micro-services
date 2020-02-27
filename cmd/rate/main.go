package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/rate"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))
	consulAddr := os.Getenv("consulAddr")
	jaegerAddr := os.Getenv("jaegerAddr")
	mongoAddr := os.Getenv("mongoAddr")
	memcAddr := os.Getenv("memcAddr")

	fmt.Printf("rate ip = %s, port = %d\n", serv_ip, serv_port)

	fmt.Printf("init rate memc with addr=%s\n", memcAddr)
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, err := tracing.Init("rate", jaegerAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := rate.InitializeDatabase(mongoAddr)
	if err != nil {
		panic(err)
	}

	srv := &rate.Server{
		Tracer:       tracer,
		Registry:     registry,
		RegCheckPort: consul_check_port,
		Port:         serv_port,
		IpAddr:       serv_ip,
		MongoClient:  mongoClient,
		MemcClient:   memc_client,
	}
	log.Fatal(srv.Run())
}
