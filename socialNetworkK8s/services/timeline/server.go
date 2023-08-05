package timeline

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
	"socialnetworkk8/services/post"
	"socialnetworkk8/dialer"
	"socialnetworkk8/services/timeline/proto"
	postpb "socialnetworkk8/services/post/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	TIMELINE_SRV_NAME = "srv-timeline"
	TIMELINE_QUERY_OK = "OK"
	TIMELINE_CACHE_PREFIX = "timeline_"
)

type TimelineSrv struct {
	proto.UnimplementedTimelineServer 
	uuid         string
	cachec       *cacheclnt.CacheClnt
	mongoCo      *mongo.Collection
	postc        postpb.PostStorageClient
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
}

func MakeTimelineSrv() *TimelineSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, mongoUrl := registry.RegisterByConfig("Timeline")
	cachec := cacheclnt.MakeCacheClnt() 
	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	collection := mongoClient.Database("socialnetwork").Collection("timeline")
	indexModel := mongo.IndexModel{Keys: bson.D{{"userid", 1}}}
	collection.Indexes().CreateOne(context.TODO(), indexModel)
	return &TimelineSrv{Port:serv_port, IpAddr:serv_ip, Tracer:tracer,
		Registry:registry, cachec:cachec, mongoCo:collection}
}

// Run starts the server
func (tlsrv *TimelineSrv) Run() error {
	if tlsrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	conn, err := dialer.Dial(post.POST_SRV_NAME, tlsrv.Registry.Client, dialer.WithTracer(tlsrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	tlsrv.postc = postpb.NewPostStorageClient(conn)
	tlsrv.uuid = uuid.New().String()
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterTimelineServer(grpcSrv, tlsrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", tlsrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = tlsrv.Registry.Register(TIMELINE_SRV_NAME, tlsrv.uuid, tlsrv.IpAddr, tlsrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

func (tlsrv *TimelineSrv) WriteTimeline(
		ctx context.Context, req *proto.WriteTimelineRequest) (
		*proto.WriteTimelineResponse, error) {
	res := &proto.WriteTimelineResponse{Ok: "No"}
	_, err := tlsrv.mongoCo.UpdateOne(
		context.TODO(), &bson.M{"userid": req.Userid}, 
		&bson.M{"$push": bson.M{"postids": req.Postid, "timestamps": req.Timestamp}},
		options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	res.Ok = TIMELINE_QUERY_OK
	key := TIMELINE_CACHE_PREFIX + strconv.FormatInt(req.Userid, 10)
	if !tlsrv.cachec.Delete(ctx, key) {
		log.Error().Msgf("cannot delete timeline of %v", key)
	}
	return res, nil
}

func (tlsrv *TimelineSrv) ReadTimeline(
		ctx context.Context, req *proto.ReadTimelineRequest) (
		*proto.ReadTimelineResponse, error) {
	res := &proto.ReadTimelineResponse{Ok: "No"}
	timeline, err := tlsrv.getUserTimeline(ctx, req.Userid)
	if err != nil {
		return nil, err
	}
	if timeline == nil {
		res.Ok = "No timeline item"
		return res, nil
	}
	start, stop, nItems := req.Start, req.Stop, int32(len(timeline.Postids))
	if start >= int32(nItems) || start >= stop {
		res.Ok = fmt.Sprintf("Cannot process start=%v end=%v for %v items", start, stop, nItems)
		return res, nil
	}	
	if stop > nItems {
		stop = nItems
	}
	postids := make([]int64, stop-start)
	for i := start; i < stop; i++ {
		postids[i-start] = timeline.Postids[nItems-i-1]
	}
	readPostReq := &postpb.ReadPostsRequest{Postids: postids}
	readPostRes, err := tlsrv.postc.ReadPosts(ctx, readPostReq)
	if err != nil {
		return nil, err 
	}
	res.Ok = readPostRes.Ok
	res.Posts = readPostRes.Posts
	return res, nil
}

func (tlsrv *TimelineSrv) getUserTimeline(ctx context.Context, userid int64) (*Timeline, error) {
	key := TIMELINE_CACHE_PREFIX + strconv.FormatInt(userid, 10) 
	timeline := &Timeline{}
	if timelineItem, err := tlsrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Debug().Msgf("Timeline %v cache miss", key)
		err = tlsrv.mongoCo.FindOne(context.TODO(), &bson.M{"userid": userid}).Decode(&timeline)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, nil
			}
			return nil, err
		} 
		log.Debug().Msgf("Found timeline %v in DB: %v", userid, timeline)
		encodedTimeline, err := json.Marshal(timeline)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		tlsrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedTimeline})
	} else {
		log.Debug().Msgf("Found timeline %v in cache!", userid)
		json.Unmarshal(timelineItem.Value, timeline)
	}
	return timeline, nil
}

type Timeline struct {
	Userid     int64   `bson:userid`
	Postids    []int64 `bson:postids`
	Timestamps []int64 `bson:timestamps`
}

