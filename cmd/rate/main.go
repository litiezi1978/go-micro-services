package main

import (
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
	host_ip := os.Getenv("hostIP")
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))
	consulPort := os.Getenv("consulPort")
	jaegerPort := os.Getenv("jaegerPort")

	consulAddr := host_ip + ":" + consulPort
	log.Printf("initing consul client with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init consul, err=%v", err)
	}

	memcAddr, err := registry.FindService("srv-memc-rate")
	if err != nil {
		log.Fatalf("failed to search srv-memc-rate from Consul, %v", err)
	}
	log.Printf("init rate memc with addr=%s\n", memcAddr)
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	jaegerAddr := host_ip + ":" + jaegerPort
	log.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("rate", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, err=%v", err)
	}
	defer closer.Close()

	mongoAddr, err := registry.FindService("srv-mongo-rate")
	if err != nil {
		log.Fatalf("failed to search srv-mongo-rate from Consul, %v", err)
	}
	log.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := rate.InitializeDatabase(mongoAddr)
	if err != nil {
		log.Fatalf("failed to init mongo, err=%v", err)
	}

	log.Printf("rate ip = %s, port = %d\n", serv_ip, serv_port)
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
