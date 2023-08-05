package post

import (
	"encoding/json"
	"strconv"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net"
	"github.com/google/uuid"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/tls"
	"socialnetworkk8/services/post/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	POST_SRV_NAME = "srv-post"
	POST_QUERY_OK = "OK"
	POST_CACHE_PREFIX = "post_"
)

type PostSrv struct {
	proto.UnimplementedPostStorageServer 
	uuid         string
	cachec       *cacheclnt.CacheClnt
	mongoCo      *mongo.Collection
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
}

func MakePostSrv() *PostSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, mongoUrl := registry.RegisterByConfig("Post")

	cachec := cacheclnt.MakeCacheClnt() 

	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	collection := mongoClient.Database("socialnetwork").Collection("post")
	indexModel := mongo.IndexModel{Keys: bson.D{{"postid", 1}}}
	collection.Indexes().CreateOne(context.TODO(), indexModel)
	return &PostSrv{Port:serv_port, IpAddr:serv_ip, Tracer:tracer, Registry:registry, cachec:cachec, mongoCo: collection}
}

// Run starts the server
func (psrv *PostSrv) Run() error {
	if psrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	psrv.uuid = uuid.New().String()
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterPostStorageServer(grpcSrv, psrv)
	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", psrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = psrv.Registry.Register(POST_SRV_NAME, psrv.uuid, psrv.IpAddr, psrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

func (psrv *PostSrv) StorePost(
		ctx context.Context, req *proto.StorePostRequest) (*proto.StorePostResponse, error) {
	res := &proto.StorePostResponse{}
	res.Ok = "No"
	postBson := postToBson(req.Post)
	if _, err := psrv.mongoCo.InsertOne(context.TODO(), postBson); err != nil {
		log.Error().Msg(err.Error())
		return res, err
	}
	res.Ok = POST_QUERY_OK
	return res, nil
}

func (psrv *PostSrv) ReadPosts(
		ctx context.Context, req *proto.ReadPostsRequest) (*proto.ReadPostsResponse, error) {
	res := &proto.ReadPostsResponse{}
	res.Ok = "No."
	posts := make([]*proto.Post, len(req.Postids))
	missing := false
	for idx, postid := range req.Postids {
		postBson, err := psrv.getPost(ctx, postid)
		if err != nil {
			return nil, err
		} 
		if postBson == nil {
			missing = true
			res.Ok = res.Ok + fmt.Sprintf(" Missing %v.", postid)
		} else {
			posts[idx] = bsonToPost(postBson)
		}
	}
	res.Posts = posts
	if !missing {
		res.Ok = POST_QUERY_OK
	}
	return res, nil
}

func (psrv *PostSrv) getPost(ctx context.Context, postid int64) (*PostBson, error) {
	key := POST_CACHE_PREFIX + strconv.FormatInt(postid, 10) 
	postBson := &PostBson{}
	if postItem, err := psrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Debug().Msgf("Post %v cache miss", key)
		err = psrv.mongoCo.FindOne(context.TODO(), &bson.M{"postid": postid}).Decode(&postBson)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, nil
			}
			return nil, err
		} 
		log.Debug().Msgf("Found post %v in DB: %v", postid, postBson)
		encodedPost, err := json.Marshal(postBson)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		psrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedPost})
	} else {
		log.Debug().Msgf("Found post %v in cache!", postid)
		json.Unmarshal(postItem.Value, postBson)
	}
	return postBson, nil
}

func postToBson(post *proto.Post) *PostBson {
	return &PostBson{
		Postid: post.Postid,
		Posttype: int32(post.Posttype),
		Timestamp: post.Timestamp,
		Creator: post.Creator,
		CreatorUname: post.Creatoruname,
		Text: post.Text,
		Usermentions: post.Usermentions,
		Medias: post.Medias,
		Urls: post.Urls,
	}
}

func bsonToPost(bson *PostBson) *proto.Post {
	return &proto.Post{
		Postid: bson.Postid,
		Posttype: proto.POST_TYPE(bson.Posttype),
		Timestamp: bson.Timestamp,
		Creator: bson.Creator,
		Creatoruname: bson.CreatorUname,
		Text: bson.Text,
		Usermentions: bson.Usermentions,
		Medias: bson.Medias,
		Urls: bson.Urls,
	}
}

type PostBson struct {
	Postid int64         `bson:postid`
	Posttype int32       `bson:posttype`
	Timestamp int64      `bson:timestamp`
	Creator int64        `bson:creator`
	CreatorUname string  `bson:creatoruname`
	Text string          `bson:text`
	Usermentions []int64 `bson:usermentions`
	Medias []int64       `bson:medias`
	Urls []string        `bson:urls`
}


