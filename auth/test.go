package auth

import (
	_ "github.com/go-sql-driver/mysql"
	pb "github.com/xjayleex/kauloud/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"time"
)

func test() {

	expires := time.Second * 3600
	secretKeyMustBeFromFile := "secret"
	/* rs, err := NewRedisUserStore(&redis.Options{
		Addr:               "localhost:6379",
		DB:                 4,
	})

	if err != nil {
		panic(err)
	} */
	rdb, err := NewMysqlGdbc(NewMysqlDataSourceName(
		"root", "start130625", "", "localhost", "3306", "testdb"))
	if err != nil {
		panic(err)
	}
	cs := NewCandidateStore(rdb)

	rdbus := NewRDBUserStore(rdb)
	auth := NewAuthServer(cs,rdbus, NewJWTManager(secretKeyMustBeFromFile,expires))
	auth.SetUpDevLogger()
	authGRPC := NewAuthGRPC(grpc.NewServer(grpc.UnaryInterceptor(auth.verifyTokenInterceptor)), auth)
	pb.RegisterAuthServiceServer(authGRPC.Server, authGRPC)
	reflection.Register(authGRPC.Server)
	if err := authGRPC.Serve("localhost:8787"); err != nil {
		panic(err)
	}
	defer authGRPC.Close()
}

