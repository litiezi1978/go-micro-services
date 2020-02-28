package main

import (
	"github.com/opentracing/opentracing-go"
	"log"
	"os"
	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/frontend"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	servIp := os.Getenv("serverIP")
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")

	servPort, err := strconv.Atoi(os.Getenv("serverPort"))
	if err != nil {
		log.Fatalf("environment var error, %v", err)
	}

	log.Printf("Init jaeger with %s", jaegerAddr)
	tracer, closer, err := tracing.Init("frontend", jaegerAddr)
	if err != nil {
		log.Fatalf("Cannot init Jaeger tracer, err=%v", err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

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
