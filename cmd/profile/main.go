package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/profile"
	"github.com/harlow/go-micro-services/tracing"

	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

func main() {
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))
	consulAddr := os.Getenv("consulAddr")
	jaegerAddr := os.Getenv("jaegerAddr")
	mongoAddr := os.Getenv("mongoAddr")
	memcAddr := os.Getenv("memcAddr")

	fmt.Printf("profile ip = %s, port = %d\n", serv_ip, serv_port)

	fmt.Printf("profile memcached addr port = %s\n", memcAddr)
	memc_client := memcache.New(memcAddr)
	memc_client.Timeout = time.Second * 2
	memc_client.MaxIdleConns = 512

	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, err := tracing.Init("profile", jaegerAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("init mongo DB with addr: %s\n", mongoAddr)
	mongoClient, err := profile.InitializeDatabase(mongoAddr)
	if err != nil {
		panic(err)
	}

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
