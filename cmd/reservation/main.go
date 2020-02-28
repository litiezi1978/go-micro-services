package main

import (
	"fmt"
	"log"
	"os"

	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/reservation"
	"github.com/harlow/go-micro-services/tracing"

	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

func main() {
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	mongoAddr := os.Getenv("mongoAddr")
	memcAddr := os.Getenv("memcAddr")
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	fmt.Printf("init memcached with addr=%s\n", memcAddr)
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	fmt.Printf("reservation ip = %s, port = %d\n", serv_ip, serv_port)

	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("reservation", jaegerAddr)
	if err != nil {
		log.Fatalf("failed to init jaeger, err=%v", err)
	}
	defer closer.Close()

	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init consul, err=%v", err)
	}

	fmt.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := reservation.InitializeDatabase(mongoAddr)
	if err != nil {
		log.Fatalf("failed to init mongo, err=%v", err)
	}

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
