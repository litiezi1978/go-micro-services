package geo

import (
	"log"
	"testing"
)

func TestInitializeDatabase(t *testing.T) {
	_, err := initializeDatabase("192.168.31.200:30099")
	if err != nil {
		log.Printf("error, %v", err)
	}
}

func TestNewGeoIndex(t *testing.T) {
	mongoClient, err := initializeDatabase("192.168.31.200:30099")
	if err != nil {
		log.Fatalf("error, %v", err)
	}
	NewGeoIndex(mongoClient)
}
