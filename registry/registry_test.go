package registry

import (
	"log"
	"testing"
)

func TestFindService(t *testing.T) {
	consulAddr := "192.168.31.200:30085"
	log.Printf("initing consul client with addr: %s\n", consulAddr)
	c, err := NewClient(consulAddr)
	if err != nil {
		log.Fatalf("failed to init consul, err=%v", err)
	}
	addr, err := c.findService("srv-memc-rate")
	if err != nil {
		log.Fatalf("failed to get service, err=%v", err)
	}
	t.Logf("got response: %s", addr)
}
