package main

import (
	"fmt"
	"github.com/harlow/go-micro-services/registry"
	"github.com/hashicorp/consul/api"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	consulAddr := os.Getenv("CONSUL_ADDR")

	serviceName := os.Getenv("SERVICE_NAME")
	podIp := os.Getenv("POD_IP")
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	regCheckPort := os.Getenv("HEALTH_CHECK_PORT")

	log.Printf("init consul with %s", consulAddr)
	registry, err := registry.NewClient(consulAddr)
	if err != nil {
		panic(err)
	}

	checkAddr := fmt.Sprintf("http://%s:%s/ping", podIp, regCheckPort)

	reg := new(api.AgentServiceRegistration)
	reg.ID = serviceName + "-" + podIp
	reg.Name = serviceName
	reg.Port = port
	reg.Address = podIp
	reg.Check = &api.AgentServiceCheck{
		HTTP:                           checkAddr,
		Timeout:                        "3s",
		Interval:                       "5s",
		DeregisterCriticalServiceAfter: "30s",
	}
	registry.Agent().ServiceRegister(reg)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "I'm healty")
	})
	log.Fatal(http.ListenAndServe(":"+regCheckPort, nil))
}
