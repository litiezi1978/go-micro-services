package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"log"
	"os"

	"strconv"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/search"
	"github.com/harlow/go-micro-services/tracing"
)

func main() {
	serv_ip := os.Getenv("serverIP")
	jaegerAddr := os.Getenv("jaegerAddr")
	consulAddr := os.Getenv("consulAddr")
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	fmt.Printf("search ip = %s, port = %d\n", serv_ip, serv_port)

	fmt.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("search", jaegerAddr)
	if err != nil {
		log.Fatalf("error init Jaeger tracing with err=%v", err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	fmt.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("error init Consul with err=%v", err)
	}

	srv := &search.Server{
		Tracer:       tracer,
		Port:         serv_port,
		IpAddr:       serv_ip,
		Registry:     registry,
		RegCheckPort: consul_check_port,
	}
	log.Fatal(srv.Run())
}
