package main

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	pb "github.com/xjayleex/kauloud/proto"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"net"
	"os"
	"sync"
	"time"
)

// Implemented Server Interface.
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
	if req.GetId() == "" || req.GetUsername() == "" || req.GetMail() == "" || req.GetPassword() == "" {
		return nil, errors.New("malformed request")
	}
	claim := &Candidate{
		Id:      req.GetId(),
		Name:    req.GetUsername(),
		Mail:    req.GetMail(),
	}
	has, err := ag.candidateStore.Has(claim)
	if err != nil {
		return nil, errors.New("unknown error on checking candidate db")
	} else {
		if !has {
			return nil, errors.New("claim is not verified")
		}
		if claim.Registered {
			return nil, errors.New("registered already")
		}
	}


	user, err := NewUser(req.GetId(), req.GetUsername(), req.GetPassword(), req.GetMail())
	if err != nil {
		ag.Logger().Debug(err)
		return &pb.SignUpResponse{Ok: false}, err
	}

	err = ag.UserStore().Save(user)

	if err != nil {
		ag.Logger().Debug(err)
		return &pb.SignUpResponse{Ok: false}, err
	}

	claim.Registered = true

	if updated, err := ag.candidateStore.Engine().Id(claim.Id).Cols("registered").Update(claim); err != nil || updated != 1 {
		ag.Logger().Warnf("Problem on updating `register` column. Relevant user is %s", claim.String())
	} else {
		ag.Logger().Infof("Successfully registered new user : %s", claim.String())
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
	candidateStore *CandidateStore
	userStore UserStore
	jwtManager *JWTManager
	logger	*logrus.Logger
}

func NewAuthServer (candidateStore *CandidateStore, userStore UserStore, jwtManager *JWTManager) *AuthServer {
	return &AuthServer{
		candidateStore: candidateStore,
		userStore: userStore,
		jwtManager: jwtManager,
		logger: logrus.New(),
	}
}

// Getter
func (as *AuthServer) CandidateStore() *CandidateStore {
	return as.candidateStore
}

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
	if ok && r.GetAccessToken() != "" {
		_, err := as.JwtManager().VerifyToken(r.AccessToken)
		if err == nil {
			return nil, errors.New("Token not expired, yet")
		} else {
		}
	}

	//as.Logger().Debugf("%s : Before Server call.",time.Now())
	h, err := handler(ctx, req)
	//as.Logger().Debugf("%s : After Server call.",time.Now())
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
		Id: user.ID(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	return token.SignedString([]byte(manager.secretKey))
}

func (manager *JWTManager) VerifyToken(accessToken string) (*UserClaims, error) {
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

type CandidateStore struct {
	RDBStore
	// CandidateInfo 어디다 넣어야할까.. ? 일단 따로 빼는게 맞는
}

func NewCandidateStore (store RDBStore) *CandidateStore {
	return &CandidateStore{
		RDBStore: store,
	}
}

func (cs *CandidateStore) Has(candidate *Candidate) (has bool, err error){
	if cs.Engine() == nil {
		return false, errors.New("nil db engine.")
	}
	if tableExists, err := cs.Engine().IsTableExist(Candidate{}); err != nil {
		return false, err
	} else {
		if !tableExists {
			return false, errors.New("table does not exists")
		}
	}

	has, err = cs.Engine().Get(candidate)

	return has, err
}

// Never, touch this `Candidate` type code, ever, ever ,ever ...
type Candidate struct {
	Id string		`json:"id" xorm:"Varchar(255) not null pk 'id' comment('StudentCode')"`
	Name string		`json:"name" xorm:"Varchar(255) not null 'name'"`
	Mail string		`json:"mail" xorm:"Varchar(255) not null 'mail'"`
	Registered bool		`xorm:"Bool not null 'registered'" exception:"true"`
	Created time.Time	`xorm:"created" exception:"true"`
}

func (candidate *Candidate) String() string {
	return fmt.Sprintf("%s %s %s\n",candidate.Id, candidate.Name, candidate.Mail)
}

type User interface {
	// user struct must implement `User` Interface and Some `Store` Interface.
	// and, must add tags on the fields.
	encoding.BinaryMarshaler // must implement MarshalBinary
	encoding.BinaryUnmarshaler // must implement UnmarshalBinary
	IsCorrectPassword(password string) bool
	ID() string
	HASHEDPASSWORD() string
	MAIL() string
}

type user struct {
	Id					string	`json:"id" xorm:"Varchar(255) not null pk 'id' comment('StudentCode')"`
	Username			string	`json:"username" xorm:"Varchar(255) not null 'username'"`
	Hashedpassword		string	`json:"password" xorm:"Varchar(255) not null 'hashedpassword'"`
	Mail				string	`json:"mail" xorm:"Varchar(255) not null 'mail'"`
	Created				time.Time	`xorm:"created"`
}

func NewUser(id string, username string, pwd string, mail string) (*user, error ) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("Cannot Hash.")
	}

	user := &user{
		Id: id,
		Username: username,
		Hashedpassword: string(hashedPassword),
		Mail: mail,
	}
	return user, nil
}

func (u *user) ID() string {
	return u.Id
}

func (u *user) USERNAME() string {
	return u.Username
}

func (u *user) HASHEDPASSWORD() string {
	return u.Hashedpassword
}

func (u *user) MAIL() string {
	return u.Mail
}

func (u *user) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *user) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, u)
}

func (u *user) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.HASHEDPASSWORD()), []byte(password))
	return err == nil
}

func (u *user) Key() string {
	return u.ID()
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
	return rud.user.ID()
}

func (rud *RedisUserData) Value() RedisValue {
	return rud.user
}

type UserStore interface {
	Save(user User) error
	FindByKey(string) (User, error)
}

// for mysql or postre, etc...
type RDBUserStore struct {
	RDBStore
}

func NewRDBUserStore (store RDBStore) *RDBUserStore {
	return &RDBUserStore{RDBStore: store}
}

func (rdbus *RDBUserStore) Save(user User) error {
	_, err := rdbus.Engine().InsertOne(user)
	return err
}

func (rdbus *RDBUserStore) FindByKey(key string) (User, error){
	userClaimBean := &user{
		Id:             key,
	}
	if has, err := rdbus.Engine().Get(userClaimBean); err != nil {
		return nil, err
	} else {
		if has {
			return userClaimBean, nil
		} else {
			return nil, errors.New("no matched item.")
		}
	}
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

func (rus *RedisUserStore) Save (user User) error {
	rus.mtx.Lock()
	defer rus.mtx.Unlock()
	ruser := NewRedisUserData(user)
	if err := rus.setNX(ruser); err != nil {
		return err
	}
	return nil
}

func (rus *RedisUserStore) FindByKey(key string) (User, error) {
	rus.mtx.RLock()
	defer rus.mtx.RUnlock()
	rCmd, err := rus.get(key)
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