package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/profile"
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
	log.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("geo", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, with err=%v", err)
	}
	defer closer.Close()

	memcAddr, err := registry.FindService("srv-memc-profile")
	if err != nil {
		log.Fatalf("failed to search srv-memc-profile from Consul, %v", err)
	}
	log.Printf("profile memcached addr port = %s\n", memcAddr)
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	mongoAddr, err := registry.FindService("srv-mongo-profile")
	if err != nil {
		log.Fatalf("failed to search srv-mongo-profile from Consul, %v", err)
	}
	log.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := profile.InitializeDatabase(mongoAddr)
	if err != nil {
		log.Fatalf("failed to init mongo, err=%v", err)
	}

	fmt.Printf("profile ip = %s, port = %d\n", serv_ip, serv_port)
	srv := profile.Server{
		Tracer:       tracer,
		Registry:     registry,
		Port:         serv_port,
		IpAddr:       serv_ip,
		MongoClient:  mongoClient,
		MemcClient:   memc_client,
		RegCheckPort: consul_check_port,
	}
	log.Fatal(srv.Run())
}
