package main

import (
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/frontend"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	servIp := os.Getenv("serverIP")
	servPort, err := strconv.Atoi(os.Getenv("serverPort"))
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")

	if err != nil {
		log.Fatalf("environment var error, %v", err)
	}

	log.Printf("init jaeger with %s", jaegerAddr)
	tracer, err := tracing.Init("frontend", jaegerAddr)
	if err != nil {
		panic(err)
	}

	log.Printf("init consul with %s", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	log.Printf("frontend ip = %s, port = %d\n", servIp, servPort)
	srv := &frontend.Server{
		Registry: registry,
		Tracer:   tracer,
		IpAddr:   servIp,
		Port:     servPort,
	}
	log.Fatal(srv.Run())
}
