package registry

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/hashicorp/consul/api"
)

type Client struct {
	*api.Client
}

func NewClient(addr string) (*Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = addr

	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{c}, nil
}

func (c *Client) Register(name string, ip string, port int, check_port int) error {
	checkAddr := fmt.Sprintf("http://%s:%d/ping", ip, check_port)

	reg := new(api.AgentServiceRegistration)
	reg.ID = name
	reg.Name = name
	reg.Port = port
	reg.Address = ip
	reg.Check = &api.AgentServiceCheck{
		HTTP:                           checkAddr,
		Timeout:                        "3s",
		Interval:                       "5s",
		DeregisterCriticalServiceAfter: "30s",
		//GRPC:     fmt.Sprintf("%v:%v/%v", IP, r.Port, r.Service),// grpc 支持，执行健康检查的地址，service 会传到 Health.Check 函数中
	}

	go startPingService(check_port)

	return c.Agent().ServiceRegister(reg)
}

func (c *Client) Deregister(id string) error {
	return c.Agent().ServiceDeregister(id)
}

func startPingService(check_port int) {
	http.HandleFunc("/ping", pingHandler)

	log.Printf("start consul check server at port=%d", check_port)
	err := http.ListenAndServe(":"+strconv.Itoa(check_port), nil)
	if err != nil {
		fmt.Errorf("error:", err)
		panic(err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "I'm healty")
}
