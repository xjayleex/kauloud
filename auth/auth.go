package main

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	pb "github.com/xjayleex/kauloud/proto"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"net"
	"os"
	"sync"
	"time"
)
type AuthGRPC struct {
	Server *grpc.Server
	*AuthServer
}

func NewAuthGRPC (gs *grpc.Server,auth *AuthServer) *AuthGRPC {
	return &AuthGRPC{
		Server:     gs,
		AuthServer: auth,
	}
}

// Implemented Server Interface.

func (ag *AuthGRPC) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		ag.Logger().Fatalf("Failed to listen: %v", addr)
		return errors.New("Failed to listen: " + addr)
	}

	ag.Logger().Infof("Serving Auth gRPC Server :: listening on %v", addr)

	if err = ag.Server.Serve(lis); err != nil {
		return err
	}
	ag.Logger().Info("Terminating Auth gRPC Server ...")

	return nil
}

func (ag *AuthGRPC) Close() {
	ag.Server.Stop()
}


func (ag *AuthGRPC) SignUp (ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	if ag == nil {
		return nil, errors.New("Err : There is no auth server objects.")
	}

	user, err := NewUser(req.GetId(), req.GetUsername(), req.GetPassword())

	if err != nil {
		ag.Logger().Debug(err)
		return &pb.SignUpResponse{Ok: false}, err
	}

	ag.UserStore().Save(user)

	if err != nil {
		ag.Logger().Debug(err)
		return &pb.SignUpResponse{Ok: false}, err
	}

	return &pb.SignUpResponse{Ok: true}, nil
}

func (ag *AuthGRPC) SignIn (ctx context.Context, req *pb.SignInRequest) (*pb.SignInResponse, error) {
	user, err := ag.UserStore().FindByKey(req.GetId())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("Authserver : nil user.")
	} else if !user.IsCorrectPassword(req.GetPassword()) {
		return nil, ErrIncorrectInfo
	}

	token, err := ag.JwtManager().Generate(user)

	if err != nil {
		ag.Logger().Debug(err)
		return nil, err
	}

	res := &pb.SignInResponse{
		AccessToken: token,
	}

	return res, nil
}

type AuthServer struct {
	userStore UserStore
	jwtManager *JWTManager
	logger	*logrus.Logger
}

func NewAuthServer (userStore UserStore, jwtManager *JWTManager) *AuthServer {

	return &AuthServer{
		userStore: userStore,
		jwtManager: jwtManager,
		logger: logrus.New(),
	}
}

// Getter

func (as *AuthServer) UserStore() UserStore {
	return as.userStore
}

func (as *AuthServer) JwtManager() *JWTManager {
	return as.jwtManager
}

func (as *AuthServer) Logger() *logrus.Logger {
	return as.logger
}

func (as *AuthServer) SetUpLogger() {
	as.Logger().SetFormatter(&logrus.JSONFormatter{})
	as.Logger().SetOutput(os.Stdout)
}

func (as *AuthServer) SetUpDevLogger() {
	as.Logger().SetFormatter(&logrus.JSONFormatter{})
	as.Logger().SetOutput(os.Stdout)
	as.Logger().SetLevel(logrus.DebugLevel)
}

// SignIn Req -> (intercept) -> 1. token not expired. (return right after)
// 							 -> 2. token expired. or no token. (gRPC handle)
func (as *AuthServer) verifyTokenInterceptor (ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	r, ok := req.(*pb.SignInRequest)
	fmt.Println("Intercepted.")
	if ok && r.GetAccessToken() != "" {
		_, err := as.JwtManager().Verify(r.AccessToken)
		if err == nil {
			return nil, errors.New("Token not expired, yet")
		} else {
			fmt.Println("foobaryer")
		}
	}

	as.Logger().Debugf("%s : Before Server call.",time.Now())
	h, err := handler(ctx, req)
	as.Logger().Debugf("%s : After Server call.",time.Now())
	return h, err
}

type JWTManager struct {
	secretKey string
	tokenDuration time.Duration
}

type UserClaims struct {
	jwt.StandardClaims
	Id string `json:"id"`
}



func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager{
	return &JWTManager{
		secretKey: secretKey,
		tokenDuration: tokenDuration,
	}
}

func (manager *JWTManager) Generate(user User) (string, error) {
	claims := UserClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(manager.tokenDuration).Unix(),
		},
		Id: user.Id(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	return token.SignedString([]byte(manager.secretKey))
}

func (manager *JWTManager) Verify(accessToken string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error){
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, ErrUnexpectedToken
			}
			return []byte(manager.secretKey), nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}
	return claims, nil
}

type User interface {
	// user struct must implement `User` Interface and Some `Store` Interface.
	// and, must add tags on the fields.
	encoding.BinaryMarshaler // must implement MarshalBinary
	encoding.BinaryUnmarshaler // must implement UnmarshalBinary
	IsCorrectPassword(password string) bool
	Id() string
	HashedPassword() string
}


type user struct {
	ID					string	`json:"id"`
	USERNAME			string	`json:"username"`
	HASHEDPASSWORD		string	`json:"password"`
}

func NewUser(id string, username string, pwd string) (*user, error ) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("Cannot Hash.")
	}

	user := &user{
		USERNAME: username,
		ID: id,
		HASHEDPASSWORD: string(hashedPassword),
	}
	return user, nil
}

func (u *user) Id() string {
	return u.ID
}

func (u *user) Username() string {
	return u.USERNAME
}

func (u *user) HashedPassword() string {
	return u.HASHEDPASSWORD
}

func (u *user) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *user) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, u)
}

func (u *user) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword()), []byte(password))
	return err == nil
}

func (u *user) Key() string {
	return u.Id()
}

func (u *user) Value() interface{} {
	return *u // pointer 쓰는 것 맞나 ? 체크하자.
}

type RedisUserData struct {
	user User
}

func NewRedisUserData (user User) *RedisUserData {
	return &RedisUserData{user: user}
}

func (rud *RedisUserData) Key() string {
	return rud.user.Id()
}

func (rud *RedisUserData) Value() RedisValue {
	return rud.user
}

type UserStore interface {
	Save(user User) error
	FindByKey(string) (User, error)
}

type RedisUserStore struct {
	mtx *sync.RWMutex
	*RedisClient
}

func NewRedisUserStore (opts *redis.Options) (*RedisUserStore, error) {
	// verifying redis opts.
	// stubby.
	// not completed, yet.
	return &RedisUserStore{
		mtx:         &sync.RWMutex{},
		RedisClient: NewRedisClient(opts),
	}, nil
}

func (rs *RedisUserStore) Save (user User) error {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	ruser := NewRedisUserData(user)
	if err := rs.setNX(ruser); err != nil {
		return err
	}
	return nil
}

func (rs *RedisUserStore) FindByKey(key string) (User, error) {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()
	rCmd, err := rs.get(key)
	if err != nil {
		return nil, err
	}
	if marshaled, err := rCmd.Bytes(); err != nil {
		return nil, err
	} else {
		user := &user{}
		if err = json.Unmarshal(marshaled, user); err != nil {
			return nil, err
		} else {
			return user, nil
		}
	}
}