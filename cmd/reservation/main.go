package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/reservation"
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
	log.Printf("initing consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init consul, err=%v", err)
	}

	jaegerAddr := host_ip + ":" + jaegerPort
	log.Printf("initing distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("reservation", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, err=%v", err)
	}
	defer closer.Close()

	memcAddr, err := registry.FindService("srv-memc-reserve")
	if err != nil {
		log.Fatalf("failed to search srv-memc-reserve from Consul, %v", err)
	}
	log.Printf("initing memcached with addr=%s\n", memcAddr)
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	mongoAddr, err := registry.FindService("srv-mongo-reserve")
	if err != nil {
		log.Fatalf("failed to search srv-mongo-reserve from Consul, %v", err)
	}
	log.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := reservation.InitializeDatabase(mongoAddr)
	if err != nil {
		log.Fatalf("failed to init mongo, err=%v", err)
	}

	log.Printf("reservation ip = %s, port = %d\n", serv_ip, serv_port)
	srv := &reservation.Server{
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
