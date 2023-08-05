package media

import (
	"encoding/json"
	"strconv"
	"fmt"
	"sync"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/rand"
	"net"
	"github.com/google/uuid"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/tls"
	"socialnetworkk8/services/media/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	MEDIA_SRV_NAME = "srv-media"
	MEDIA_QUERY_OK = "OK"
	MEDIA_CACHE_PREFIX = "media_"
)

type MediaSrv struct {
	proto.UnimplementedMediaStorageServer 
	uuid         string
	cachec       *cacheclnt.CacheClnt
	mongoCo      *mongo.Collection
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	sid          int32 // sid is a random number between 0 and 2^30
	ucount       int32 //This server may overflow with over 2^31 medias
    mu           sync.Mutex
}

func MakeMediaSrv() *MediaSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, mongoUrl := registry.RegisterByConfig("Media")
	cachec := cacheclnt.MakeCacheClnt() 
	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	collection := mongoClient.Database("socialnetwork").Collection("media")
	indexModel := mongo.IndexModel{Keys: bson.D{{"mediaid", 1}}}
	collection.Indexes().CreateOne(context.TODO(), indexModel)
	return &MediaSrv{Port:serv_port, IpAddr:serv_ip, Tracer:tracer, Registry:registry, cachec:cachec, mongoCo:collection}
}

// Run starts the server
func (msrv *MediaSrv) Run() error {
	if msrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	//zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	log.Info().Msg("Initializing gRPC Server...")
	msrv.uuid = uuid.New().String()
	msrv.sid = rand.Int31n(536870912) // 2^29
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterMediaStorageServer(grpcSrv, msrv)

	// listener
	log.Info().Msg("Initializing request listener ...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", msrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = msrv.Registry.Register(MEDIA_SRV_NAME, msrv.uuid, msrv.IpAddr, msrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

func (msrv *MediaSrv) StoreMedia(
		ctx context.Context, req *proto.StoreMediaRequest) (*proto.StoreMediaResponse, error){
	res := &proto.StoreMediaResponse{Ok: "No"}
	mId := msrv.getNextMediaId()
	media := &Media{mId, req.Mediatype, req.Mediadata}
	if _, err := msrv.mongoCo.InsertOne(context.TODO(), media); err != nil {
		log.Error().Msg(err.Error())
		return res, err
	}
	res.Ok = MEDIA_QUERY_OK
	res.Mediaid = mId
	return res, nil
}

func (msrv *MediaSrv) ReadMedia(
		ctx context.Context, req *proto.ReadMediaRequest) (*proto.ReadMediaResponse, error){
	res := &proto.ReadMediaResponse{Ok: "No"}
	mediatypes := make([]string, len(req.Mediaids))
	mediadatas := make([][]byte, len(req.Mediaids))
	missing := false
	for idx, mediaid := range req.Mediaids {
		media, err := msrv.getMedia(ctx, mediaid)
		if err != nil {
			return nil, err
		} 
		if media == nil {
			missing = true
			res.Ok = res.Ok + fmt.Sprintf(" Missing %v.", mediaid)
		} else {
			mediatypes[idx] = media.Type
			mediadatas[idx] = media.Data
		}
	}
	res.Mediatypes = mediatypes
	res.Mediadatas = mediadatas
	if !missing {
		res.Ok = MEDIA_QUERY_OK
	}
	return res, nil
}

func (msrv *MediaSrv) getMedia(ctx context.Context, mediaid int64) (*Media, error) {
	key := MEDIA_CACHE_PREFIX + strconv.FormatInt(mediaid, 10) 
	media := &Media{}
	if mediaItem, err := msrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Info().Msgf("Media %v cache miss", key)
		err = msrv.mongoCo.FindOne(context.TODO(), &bson.M{"mediaid": mediaid}).Decode(&media);
		if  err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, nil
			}
			return nil, err
		} 
		log.Info().Msgf("Found media %v in DB: %v", mediaid, media)
		encodedMedia, err := json.Marshal(media)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		msrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedMedia})
	} else {
		log.Info().Msgf("Found media %v in cache!", mediaid)
		json.Unmarshal(mediaItem.Value, media)
	}
	return media, nil
}

type Media struct {
	Mediaid int64  `bson:mediaid`
	Type    string `bson:type`
	Data    []byte `bson:data`
}

func (msrv *MediaSrv) incCountSafe() int32 {
	msrv.mu.Lock()
	defer msrv.mu.Unlock()
	msrv.ucount++
	return msrv.ucount
}

func (msrv *MediaSrv) getNextMediaId() int64 {
	return int64(msrv.sid)*1e10 + int64(msrv.incCountSafe())
}
