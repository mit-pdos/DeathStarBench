package user

import (
	"encoding/json"
	"crypto/sha256"
	"time"
	"fmt"
	"math/rand"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net"
	"sync"
	"github.com/google/uuid"
	//"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/services/user/proto"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/tls"
	opentracing "github.com/opentracing/opentracing-go"
	"socialnetworkk8/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	USER_SRV_NAME = "srv-user"
	USER_QUERY_OK = "OK"
	USER_CACHE_PREFIX = "user_"
)

// Server implements the user service
type UserSrv struct {
	proto.UnimplementedUserServer
	uuid   		 string
	cachec       *cacheclnt.CacheClnt
	mclnt        *mongo.Client
	mongoCo      *mongo.Collection
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	sid          int32 // sid is a random number between 0 and 2^30
	ucount       int32 //This server may overflow with over 2^31 users
    mu           sync.Mutex
	cacheCounter *tracing.Counter
	checkCounter *tracing.Counter
}

func MakeUserSrv() *UserSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, mongoUrl := registry.RegisterByConfig("User")
	cachec := cacheclnt.MakeCacheClnt() 
	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	collection := mongoClient.Database("socialnetwork").Collection("user")
	indexModel := mongo.IndexModel{Keys: bson.D{{"username", 1}}}
	collection.Indexes().CreateOne(context.TODO(), indexModel)
	return &UserSrv{Port: serv_port, IpAddr: serv_ip, Tracer: tracer, Registry: registry, cachec: cachec,
		mclnt: mongoClient, mongoCo: collection, cacheCounter: tracing.MakeCounter("Cache"), checkCounter: tracing.MakeCounter("Check-User")}
}

// Run starts the server
func (usrv *UserSrv) Run() error {
	if usrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	usrv.uuid = uuid.New().String()
	usrv.sid = rand.Int31n(536870912) // 2^29
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterUserServer(grpcSrv, usrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", usrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	err = usrv.Registry.Register(USER_SRV_NAME, usrv.uuid, usrv.IpAddr, usrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return grpcSrv.Serve(lis)
}

func (usrv *UserSrv) CheckUser(
		ctx context.Context, req *proto.CheckUserRequest) (*proto.CheckUserResponse, error) {
	t0 := time.Now()
	defer usrv.checkCounter.AddTimeSince(t0)
	log.Debug().Msgf("Checking user at %v: %v", usrv.sid, req.Usernames)
	userids := make([]int64, len(req.Usernames))
	res := &proto.CheckUserResponse{}
	res.Ok = "No"
	missing := false
	for idx, username := range req.Usernames {
		user, err := usrv.getUserbyUname(ctx, username)
		if err != nil {
			return res, err
		}
		if user == nil {
			userids[idx] = int64(-1)
			missing = true
		} else {
			userids[idx] = user.Userid
		}
	}
	res.Userids = userids
	if !missing {
		res.Ok = USER_QUERY_OK
	}
	return res, nil
}

func (usrv *UserSrv) RegisterUser(
		ctx context.Context, req *proto.RegisterUserRequest) (*proto.UserResponse, error) {
	log.Debug().Msgf("Register user at %v: %v", usrv.sid, req)
	res := &proto.UserResponse{}
	res.Ok = "No"
	user, err := usrv.getUserbyUname(ctx, req.Username)
	if err != nil {
		return res, err
	}
	if user != nil {
		res.Ok = fmt.Sprintf("Username %v already exist", req.Username)
		return res, nil
	}
	pswd_hashed := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password)))
	userid := usrv.getNextUserId()
	newUser := User{
		Userid: userid,
		Username: req.Username,
		Lastname: req.Lastname,
		Firstname: req.Firstname,
		Password: pswd_hashed}
	if _, err := usrv.mongoCo.InsertOne(context.TODO(), &newUser); err != nil {
		log.Error().Msg(err.Error())
		return res, err
	}
	res.Ok = USER_QUERY_OK
	res.Userid = userid
	return res, nil
}

func (usrv *UserSrv) Login(
		ctx context.Context, req *proto.LoginRequest) (*proto.UserResponse, error) {
	log.Debug().Msgf("User login with %v: %v", usrv.sid, req)
	res := &proto.UserResponse{}
	res.Ok = "Login Failure."
	user, err := usrv.getUserbyUname(ctx, req.Username)
	if err != nil {
		return res, err
	}
	if user != nil && fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password))) == user.Password {
		res.Ok = USER_QUERY_OK
		res.Userid = user.Userid
	}
	return res, nil
}

func (usrv *UserSrv) getUserbyUname(ctx context.Context, username string) (*User, error) {
	key := USER_CACHE_PREFIX + username
	user := &User{}
	t0 := time.Now()
	userItem, err := usrv.cachec.Get(ctx, key)
	usrv.cacheCounter.AddTimeSince(t0)
	if err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Debug().Msgf("User %v cache miss", key)
		err = usrv.mongoCo.FindOne(context.TODO(), &bson.M{"username": username}).Decode(&user)
		if  err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, nil
			}
			return nil, err
		} 
		log.Debug().Msgf("Found user %v in DB: %v", username, user)
		encodedUser, err := json.Marshal(user)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		usrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedUser})
	} else {
		log.Debug().Msgf("Found user %v in cache!", username)
		json.Unmarshal(userItem.Value, user)
	}
	return user, nil
}

type User struct {
	Userid    int64  `bson:userid`
	Firstname string `bson:firstname`
	Lastname  string `bson:lastname`
	Username  string `bson:username`
	Password  string `bson:password`
}

func (usrv *UserSrv) incCountSafe() int32 {
	usrv.mu.Lock()
	defer usrv.mu.Unlock()
	usrv.ucount++
	return usrv.ucount
}

func (usrv *UserSrv) getNextUserId() int64 {
	return int64(usrv.sid)*1e10 + int64(usrv.incCountSafe())
}

