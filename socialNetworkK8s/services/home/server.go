package home

import (
	"encoding/json"
	"strconv"
	"fmt"
	"net"
	"github.com/google/uuid"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/tls"
	"socialnetworkk8/dialer"
	"socialnetworkk8/services/home/proto"
	"socialnetworkk8/services/post"
	postpb "socialnetworkk8/services/post/proto"
	"socialnetworkk8/services/graph"
	graphpb "socialnetworkk8/services/graph/proto"
	"socialnetworkk8/services/timeline"
	tlpb "socialnetworkk8/services/timeline/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	HOME_SRV_NAME = "srv-home"
	HOME_QUERY_OK = "OK"
	HOME_CACHE_PREFIX = "home_"
)

type HomeSrv struct {
	proto.UnimplementedHomeServer 
	uuid         string
	cachec       *cacheclnt.CacheClnt
	postc        postpb.PostStorageClient
	graphc       graphpb.GraphClient
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
}

func MakeHomeSrv() *HomeSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, _ := registry.RegisterByConfig("Home")
	cachec := cacheclnt.MakeCacheClnt() 
	return &HomeSrv{Port:serv_port, IpAddr:serv_ip, Tracer:tracer, Registry:registry, cachec:cachec}
}

// Run starts the server
func (hsrv *HomeSrv) Run() error {
	if hsrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	postConn, err := dialer.Dial( post.POST_SRV_NAME, hsrv.Registry.Client, dialer.WithTracer(hsrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	hsrv.postc = postpb.NewPostStorageClient(postConn)
	graphConn, err := dialer.Dial(graph.GRAPH_SRV_NAME, hsrv.Registry.Client, dialer.WithTracer(hsrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	hsrv.graphc = graphpb.NewGraphClient(graphConn)
	hsrv.uuid = uuid.New().String()
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterHomeServer(grpcSrv, hsrv)
	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", hsrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = hsrv.Registry.Register(HOME_SRV_NAME, hsrv.uuid, hsrv.IpAddr, hsrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

func (hsrv *HomeSrv) WriteHomeTimeline(
		ctx context.Context, req *proto.WriteHomeTimelineRequest) (
		*tlpb.WriteTimelineResponse, error) {
	res := &tlpb.WriteTimelineResponse{Ok: "No"}
	otherUserIds := make(map[int64]bool, 0)
	argFollower := &graphpb.GetFollowersRequest{Followeeid: req.Userid}
	resFollower, err := hsrv.graphc.GetFollowers(ctx, argFollower)
	if err != nil {
		return nil, err
	}
	for _, followerid := range resFollower.Userids {
		otherUserIds[followerid] = true
	}
	for _, mentionid := range req.Usermentionids {
		otherUserIds[mentionid] = true
	}
	log.Debug().Msgf("Updating timeline for %v users", len(otherUserIds))
	missing := false
	for userid := range otherUserIds {
		hometl, err := hsrv.getHomeTimeline(ctx, userid)
		if err != nil {
			res.Ok = res.Ok + fmt.Sprintf(" Error getting home timeline for %v.", userid)	
			missing = true
			continue
		}
		hometl.Postids = append(hometl.Postids, req.Postid)	
		hometl.Timestamps = append(hometl.Timestamps, req.Timestamp)	
		key := HOME_CACHE_PREFIX + strconv.FormatInt(userid, 10) 
		encodedHometl, err := json.Marshal(hometl)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		hsrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedHometl})
	}
	if !missing {
		res.Ok = HOME_QUERY_OK
	}
	return res, nil 
}

func (hsrv *HomeSrv) ReadHomeTimeline(
		ctx context.Context, req *tlpb.ReadTimelineRequest) (*tlpb.ReadTimelineResponse, error) {
	res := &tlpb.ReadTimelineResponse{Ok: "No"}
	timeline, err := hsrv.getHomeTimeline(ctx, req.Userid)
	if err != nil {
		return nil, err
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
	readPostRes, err := hsrv.postc.ReadPosts(ctx, readPostReq)
	if err != nil {
		return nil, err 
	}
	res.Ok = readPostRes.Ok
	res.Posts = readPostRes.Posts
	return res, nil
}

func (hsrv *HomeSrv) getHomeTimeline(ctx context.Context, userid int64) (*timeline.Timeline, error) {
	key := HOME_CACHE_PREFIX + strconv.FormatInt(userid, 10) 
	timeline := &timeline.Timeline{}
	if timelineItem, err := hsrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Debug().Msgf("Home timeline %v cache miss", key)
		timeline.Userid = userid
	} else {
		json.Unmarshal(timelineItem.Value, timeline)
		log.Debug().Msgf("Found home timeline %v in cache! %v", userid, timeline)
	}
	return timeline, nil
}
