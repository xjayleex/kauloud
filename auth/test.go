package main

import (
	"github.com/go-redis/redis/v8"
	pb "github.com/xjayleex/kauloud/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"time"
)

func main() {

	expires := time.Second * 3600
	secretKeyMustBeFromFile := "secret"
	rs, err := NewRedisUserStore(&redis.Options{
		Addr:               "localhost:6379",
		DB:                 4,
	})

	if err != nil {
		panic(err)
	}
	NewJWTManager(secretKeyMustBeFromFile,expires)
	auth := NewAuthServer(rs, NewJWTManager(secretKeyMustBeFromFile,expires))
	auth.SetUpDevLogger()
	authGRPC := NewAuthGRPC(grpc.NewServer(grpc.UnaryInterceptor(auth.verifyTokenInterceptor)), auth)
	pb.RegisterAuthServiceServer(authGRPC.Server, authGRPC)
	reflection.Register(authGRPC.Server)
	if err := authGRPC.Serve("localhost:8787"); err != nil {
		panic(err)
	}
	defer authGRPC.Close()
}

