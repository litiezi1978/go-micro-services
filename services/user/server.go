package user

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/user/proto"
	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const name = "srv-user"

type Server struct {
	users map[string]string

	Tracer       opentracing.Tracer
	Registry     *registry.Client
	Port         int
	IpAddr       string
	MongoClient  *mongo.Client
	RegCheckPort int
}

func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	if s.users == nil {
		s.users = loadUsers(s.MongoClient)
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(s.Tracer)),
	)

	pb.RegisterUserServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = s.Registry.Register(name, s.IpAddr, s.Port, s.RegCheckPort)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}

	return srv.Serve(lis)
}

func (s *Server) Shutdown() {
	s.Registry.Deregister(name)
}

func (s *Server) CheckUser(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	span := opentracing.SpanFromContext(ctx)
	span.LogKV("username", req.Username,
		"password", req.Password)

	res := new(pb.Result)

	sum := sha256.Sum256([]byte(req.Password))
	pass := fmt.Sprintf("%x", sum)

	res.Correct = false
	if true_pass, found := s.users[req.Username]; found {
		res.Correct = pass == true_pass
	}
	span.LogKV("checkResult", res.Correct)

	return res, nil
}

func loadUsers(mongoClient *mongo.Client) map[string]string {
	collection := mongoClient.Database("user-db").Collection("user")

	var users []User
	ctx, _ := context.WithCancel(context.Background())
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatal("failed to get cursor", err)
	}
	err = cursor.All(ctx, &users)
	if err != nil {
		log.Fatal("failed to read cursor", err)
	}

	res := make(map[string]string)
	for _, user := range users {
		res[user.Username] = user.Password
	}

	fmt.Printf("Done load users\n")
	return res
}
