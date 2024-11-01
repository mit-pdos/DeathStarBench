package user

import (
	"encoding/json"
	"crypto/sha256"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"time"
	"fmt"
	"math/rand"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net"
	"sync"
	"net/http"
	"net/http/pprof"
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
	"google.golang.org/grpc/keepalive"
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
	dbCounter    *tracing.Counter
	cacheCounter *tracing.Counter
	loginCounter *tracing.Counter
	checkCounter *tracing.Counter
}

func MakeUserSrv() *UserSrv {
	tune.Init()
	log.Info().Msg("Reading config...")
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Error().Msgf("Got error while reading config: %v", err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]string
	json.Unmarshal([]byte(byteValue), &result)

	log.Info().Msg("Successfull")

	serv_port, _ := strconv.Atoi(result["UserPort"])
	serv_ip := result["UserIP"]

	log.Info().Msgf("Read target port: %v", serv_port)
	log.Info().Msgf("Read consul address: %v", result["consulAddress"])
	log.Info().Msgf("Read jaeger address: %v", result["jaegerAddress"])
	var (
		jaegeraddr = flag.String("jaegeraddr", result["jaegerAddress"], "Jaeger address")
		consuladdr = flag.String("consuladdr", result["consulAddress"], "Consul address")
	)
	flag.Parse()

	log.Info().Msgf("Initializing jaeger [service name: %v | host: %v]...", "user", *jaegeraddr)
	tracer, err := tracing.Init("user", *jaegeraddr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing jaeger agent: %v", err)
	}
	log.Info().Msg("Jaeger agent initialized")

	log.Info().Msgf("Initializing consul agent [host: %v]...", *consuladdr)
	registry, err := registry.NewClient(*consuladdr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing consul agent: %v", err)
	}
	log.Info().Msg("Consul agent initialized")
	log.Info().Msg("Start cache and DB connections")
	cachec := cacheclnt.MakeCacheClnt() 

	mongoUrl := "mongodb://" + result["MongoAddress"]
	log.Info().Msgf("Read database URL: %v", mongoUrl)
	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	collection := mongoClient.Database("socialnetwork").Collection("user")
	indexModel := mongo.IndexModel{Keys: bson.D{{"username", 1}}}
	name, err := collection.Indexes().CreateOne(context.TODO(), indexModel)
	log.Info().Msgf("Name of index created: %v", name)
	log.Info().Msg("New mongo session successfull...")
	return &UserSrv{
		Port:         serv_port,
		IpAddr:       serv_ip,
		Tracer:       tracer,
		Registry:     registry,
		cachec:       cachec,
		mclnt:        mongoClient,
		mongoCo:      collection,
		dbCounter:    tracing.MakeCounter("DB"),
		cacheCounter: tracing.MakeCounter("Cache"),
		loginCounter: tracing.MakeCounter("Login"),
		checkCounter: tracing.MakeCounter("Check-User"),
	}
}

// Run starts the server
func (usrv *UserSrv) Run() error {
	if usrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	usrv.uuid = uuid.New().String()
	usrv.sid = rand.Int31n(536870912) // 2^29
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		//grpc.UnaryInterceptor(
		//	otgrpc.OpenTracingServerInterceptor(usrv.Tracer),
		//),
	}

	if tlsopt := tls.GetServerOpt(); tlsopt != nil {
		opts = append(opts, tlsopt)
	}

	grpcSrv := grpc.NewServer(opts...)

	proto.RegisterUserServer(grpcSrv, usrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", usrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	http.Handle("/pprof/cpu", http.HandlerFunc(pprof.Profile))
	go func() {
		log.Error().Msgf("Error ListenAndServe: %v", http.ListenAndServe(":5000", nil))
	}()

	err = usrv.Registry.Register(USER_SRV_NAME, usrv.uuid, usrv.IpAddr, usrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return grpcSrv.Serve(lis)
}

// Shutdown cleans up any processes
func (usrv *UserSrv) Shutdown() {
	usrv.mclnt.Disconnect(context.Background())
	usrv.Registry.Deregister(usrv.uuid)
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
	t0 := time.Now()
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
	usrv.loginCounter.AddTimeSince(t0)
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
		t1 := time.Now()
		err = usrv.mongoCo.FindOne(context.TODO(), &bson.M{"username": username}).Decode(&user)
		usrv.dbCounter.AddTimeSince(t1)
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

