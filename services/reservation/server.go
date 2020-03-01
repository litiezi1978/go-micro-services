package reservation

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/reservation/proto"
	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/mgo.v2/bson"
)

const name = "srv-reservation"

type Server struct {
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	MongoClient  *mongo.Client
	Registry     *registry.Client
	MemcClient   *memcache.Client
	RegCheckPort int
}

func (s *Server) Run() error {
	if s.Port == 0 {
		log.Fatal("server port must be set")
	}

	grpcSrv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(s.Tracer)),
	)
	pb.RegisterReservationServer(grpcSrv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = s.Registry.Register(name, s.IpAddr, s.Port, s.RegCheckPort)
	if err != nil {
		log.Fatalf("failed register: %v", err)
	}

	return grpcSrv.Serve(lis)
}

func (s *Server) Shutdown() {
	s.Registry.Deregister(name)
}

func (s *Server) MakeReservation(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	db := s.MongoClient.Database("reservation-db")
	collReserve := db.Collection("reservation")
	collNumber := db.Collection("number")

	res := new(pb.Result)
	res.HotelId = make([]string, 0)

	inDate, _ := time.Parse(time.RFC3339, req.InDate+"T12:00:00+00:00")
	outDate, _ := time.Parse(time.RFC3339, req.OutDate+"T12:00:00+00:00")

	hotelId := req.HotelId[0]

	indate := inDate.String()[0:10]

	memc_date_num_map := make(map[string]int)

	for inDate.Before(outDate) {
		// check reservations
		count := 0
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]

		// first check memc
		memc_key := hotelId + "_" + inDate.String()[0:10] + "_" + outdate
		item, err := s.MemcClient.Get(memc_key)
		if err == nil {
			// memcached hit
			count, _ = strconv.Atoi(string(item.Value))
			fmt.Printf("memcached hit %s = %d\n", memc_key, count)
			memc_date_num_map[memc_key] = count + int(req.RoomNumber)

		} else if err == memcache.ErrCacheMiss {
			// memcached miss
			fmt.Printf("memcached miss\n")

			reserve := make([]reservation, 0)
			cursor, err := collReserve.Find(ctx, &bson.M{"hotelId": hotelId, "inDate": indate, "outDate": outdate})
			if err != nil {
				panic(err)
			}
			err = cursor.All(ctx, &reserve)
			if err == mongo.ErrNoDocuments {
				for _, r := range reserve {
					count += r.Number
				}
				memc_date_num_map[memc_key] = count + int(req.RoomNumber)
			}

		} else {
			fmt.Printf("Memmcached error = %s\n", err)
			panic(err)
		}

		// check capacity
		// check memc capacity
		memc_cap_key := hotelId + "_cap"
		item, err = s.MemcClient.Get(memc_cap_key)
		hotel_cap := 0
		if err == nil {
			// memcached hit
			hotel_cap, _ = strconv.Atoi(string(item.Value))
			fmt.Printf("memcached hit %s = %d\n", memc_cap_key, hotel_cap)
		} else if err == memcache.ErrCacheMiss {
			// memcached miss
			var num number

			err = collNumber.FindOne(ctx, &bson.M{"hotelId": hotelId}).Decode(&num)
			if err == nil {
				hotel_cap = int(num.Number)

				// write to memcache
				s.MemcClient.Set(&memcache.Item{Key: memc_cap_key, Value: []byte(strconv.Itoa(hotel_cap))})
			}

		} else {
			fmt.Printf("Memmcached error = %s\n", err)
			panic(err)
		}

		if count+int(req.RoomNumber) > hotel_cap {
			return res, nil
		}
		indate = outdate
	}

	// only update reservation number cache after check succeeds
	for key, val := range memc_date_num_map {
		s.MemcClient.Set(&memcache.Item{Key: key, Value: []byte(strconv.Itoa(val))})
	}

	inDate, _ = time.Parse(time.RFC3339, req.InDate+"T12:00:00+00:00")

	indate = inDate.String()[0:10]

	for inDate.Before(outDate) {
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]
		my_reserv := reservation{
			HotelId:      hotelId,
			CustomerName: req.CustomerName,
			InDate:       indate,
			OutDate:      outdate,
			Number:       int(req.RoomNumber)}

		_, err := collReserve.InsertOne(ctx, &my_reserv)
		if err != nil {
			log.Fatalf("insert reserve error: %v", err)
		}
		indate = outdate
	}

	res.HotelId = append(res.HotelId, hotelId)
	return res, nil
}

func (s *Server) CheckAvailability(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	res := new(pb.Result)
	res.HotelId = make([]string, 0)

	db := s.MongoClient.Database("reservation-db")
	coll_reserve := db.Collection("reservation")
	coll_number := db.Collection("number")

	for _, hotelId := range req.HotelId {
		log.Printf("reservation check hotel %s\n", hotelId)
		inDate, _ := time.Parse(time.RFC3339, req.InDate+"T12:00:00+00:00")
		outDate, _ := time.Parse(time.RFC3339, req.OutDate+"T12:00:00+00:00")

		indateYMD := inDate.String()[0:10]

		for inDate.Before(outDate) {
			// check reservations
			count := 0
			inDate = inDate.AddDate(0, 0, 1)
			log.Printf("reservation check date %s\n", inDate.String()[0:10])
			outdateYMD := inDate.String()[0:10]

			// first check memc
			memc_key := hotelId + "_" + inDate.String()[0:10] + "_" + outdateYMD
			item, err := s.MemcClient.Get(memc_key)

			if err == nil {
				// memcached hit
				count, _ = strconv.Atoi(string(item.Value))
				log.Printf("memcached hit %s = %d\n", memc_key, count)
			} else if err == memcache.ErrCacheMiss {
				// memcached miss
				reserve := make([]reservation, 0)

				cursor, err := coll_reserve.Find(ctx, &bson.M{"hotelId": hotelId, "inDate": indateYMD, "outDate": outdateYMD})
				if err != nil {
					log.Fatalf("failed to read table researve, error is %v", err)
				}
				err = cursor.All(ctx, &reserve)
				if len(reserve) > 0 {
					for _, r := range reserve {
						fmt.Printf("reservation check reservation number = %s\n", hotelId)
						count += r.Number
					}
					// update memcached
					s.MemcClient.Set(&memcache.Item{Key: memc_key, Value: []byte(strconv.Itoa(count))})
				}
			} else {
				log.Fatalf("Memmcached error = %s\n", err)
			}

			// check capacity
			// check memc capacity
			memc_cap_key := hotelId + "_cap"
			item, err = s.MemcClient.Get(memc_cap_key)
			hotel_cap := 0

			if err == nil {
				// memcached hit
				hotel_cap, _ = strconv.Atoi(string(item.Value))
				fmt.Printf("memcached hit %s = %d\n", memc_cap_key, hotel_cap)

			} else if err == memcache.ErrCacheMiss {
				var num number

				err = coll_number.FindOne(ctx, &bson.M{"hotelId": hotelId}).Decode(&num)
				if err == nil {
					hotel_cap = int(num.Number)
					// update memcached
					s.MemcClient.Set(&memcache.Item{Key: memc_cap_key, Value: []byte(strconv.Itoa(hotel_cap))})
				}
			} else {
				log.Fatalf("Memmcached error = %s\n", err)
			}

			if count+int(req.RoomNumber) > hotel_cap {
				break
			}
			indateYMD = outdateYMD

			if inDate.Equal(outDate) {
				res.HotelId = append(res.HotelId, hotelId)
			}
		}
	}

	return res, nil
}
