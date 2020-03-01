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
	host_ip := os.Getenv("hostIP")
	serv_ip := os.Getenv("serverIP")
	serv_port, _ := strconv.Atoi(os.Getenv("serverPort"))
	jaegerPort := os.Getenv("jaegerPort")
	consulPort := os.Getenv("consulPort")
	consul_check_port, _ := strconv.Atoi(os.Getenv("consulCheckPort"))

	jaegerAddr := host_ip + ":" + jaegerPort
	log.Printf("init distributed tracing with addr: %s\n", jaegerAddr)
	tracer, closer, err := tracing.Init("search", jaegerAddr)
	if err != nil {
		log.Fatalf("error init Jaeger tracing with err=%v", err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	consulAddr := host_ip + ":" + consulPort
	log.Printf("init consul with addr: %s\n", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("error init Consul with err=%v", err)
	}

	fmt.Printf("search ip = %s, port = %d\n", serv_ip, serv_port)
	srv := &search.Server{
		Tracer:       tracer,
		Port:         serv_port,
		IpAddr:       serv_ip,
		Registry:     registry,
		RegCheckPort: consul_check_port,
	}
	log.Fatal(srv.Run())
}
