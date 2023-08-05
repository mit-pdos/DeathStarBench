package graph

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
	"socialnetworkk8/dialer"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/tls"
	"socialnetworkk8/services/user"
	"socialnetworkk8/services/graph/proto"
	userpb "socialnetworkk8/services/user/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	GRAPH_SRV_NAME = "srv-graph"
	GRAPH_QUERY_OK = "OK"
	FOLLOWER_CACHE_PREFIX = "followers_"
	FOLLOWEE_CACHE_PREFIX = "followees_"
)

// Server implements the user service
type GraphSrv struct {
	proto.UnimplementedGraphServer 
	uuid         string
	cachec       *cacheclnt.CacheClnt
	mongoFlwERCo *mongo.Collection
	mongoFlwEECo *mongo.Collection
	userc        userpb.UserClient
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
}

func MakeGraphSrv() *GraphSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, mongoUrl := registry.RegisterByConfig("Graph")
	cachec := cacheclnt.MakeCacheClnt() 
	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	followersCo := mongoClient.Database("socialnetwork").Collection("graph-follower")
	followeesCo := mongoClient.Database("socialnetwork").Collection("graph-followee")
	indexModel := mongo.IndexModel{Keys: bson.D{{"userid", 1}}}
	followersCo.Indexes().CreateOne(context.TODO(), indexModel)
	followeesCo.Indexes().CreateOne(context.TODO(), indexModel)
	return &GraphSrv{Port: serv_port, IpAddr: serv_ip, Tracer: tracer, Registry: registry,
		cachec: cachec, mongoFlwERCo: followersCo, mongoFlwEECo: followeesCo}
}

// Run starts the server
func (gsrv *GraphSrv) Run() error {
	if gsrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	conn, err := dialer.Dial(user.USER_SRV_NAME, gsrv.Registry.Client, dialer.WithTracer(gsrv.Tracer))
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	gsrv.userc = userpb.NewUserClient(conn)
	gsrv.uuid = uuid.New().String()
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterGraphServer(grpcSrv, gsrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", gsrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = gsrv.Registry.Register(GRAPH_SRV_NAME, gsrv.uuid, gsrv.IpAddr, gsrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}

func (gsrv *GraphSrv) GetFollowers(
		ctx context.Context, req *proto.GetFollowersRequest) (*proto.GraphGetResponse, error) {
	res := &proto.GraphGetResponse{}
	res.Ok = "No"
	res.Userids = make([]int64, 0)
	followers, err := gsrv.getFollowers(ctx, req.Followeeid)
	if err == nil {
		res.Userids = followers
		res.Ok = GRAPH_QUERY_OK
	}
	return res, nil
}

func (gsrv *GraphSrv) GetFollowees(
		ctx context.Context, req *proto.GetFolloweesRequest) (*proto.GraphGetResponse, error) {
	res := &proto.GraphGetResponse{}
	res.Ok = "No"
	res.Userids = make([]int64, 0)
	followees, err := gsrv.getFollowees(ctx, req.Followerid)
	if err == nil {
		res.Userids = followees
		res.Ok = GRAPH_QUERY_OK
	}
	return res, nil
}

func (gsrv *GraphSrv) Follow(
		ctx context.Context, req *proto.FollowRequest) (*proto.GraphUpdateResponse, error) {
	return gsrv.updateGraph(ctx, req.Followerid, req.Followeeid, true)
}

func (gsrv *GraphSrv) Unfollow(
		ctx context.Context, req *proto.UnfollowRequest) (*proto.GraphUpdateResponse, error) {
	return gsrv.updateGraph(ctx, req.Followerid, req.Followeeid, false)
}

func (gsrv *GraphSrv) FollowWithUname(
		ctx context.Context, req *proto.FollowWithUnameRequest) (*proto.GraphUpdateResponse, error) {
	return gsrv.updateGraphWithUname(ctx, req.Followeruname, req.Followeeuname, true)
}

func (gsrv *GraphSrv) UnfollowWithUname(
		ctx context.Context, req *proto.UnfollowWithUnameRequest)(*proto.GraphUpdateResponse, error){
	return gsrv.updateGraphWithUname(ctx, req.Followeruname, req.Followeeuname, false)
}

func (gsrv *GraphSrv) updateGraph(
		ctx context.Context, followerid, followeeid int64, isFollow bool) (
		*proto.GraphUpdateResponse, error) {
	res := &proto.GraphUpdateResponse{}
	res.Ok = "No"
	log.Debug().Msgf("Updating graph. %v follows %v; Add edge? %v", followerid, followeeid, isFollow)
	if followerid == followeeid {
		if isFollow {
			res.Ok = "Cannot follow self."
		} else {
			res.Ok = "Cannot unfollow self."
		}
		return res, nil
	}
	var err1, err2 error
	if isFollow {
		_, err1 = gsrv.mongoFlwERCo.UpdateOne(
			context.TODO(), &bson.M{"userid": followeeid}, 
			&bson.M{"$addToSet": bson.M{"edges": followerid}}, options.Update().SetUpsert(true))
		_, err2 = gsrv.mongoFlwEECo.UpdateOne(
			context.TODO(), &bson.M{"userid": followerid}, 
			&bson.M{"$addToSet": bson.M{"edges": followeeid}}, options.Update().SetUpsert(true))
	} else {
		_, err1 = gsrv.mongoFlwERCo.UpdateOne(
			context.TODO(), &bson.M{"userid": followeeid}, 
			&bson.M{"$pull": bson.M{"edges": followerid}})
		_, err2 = gsrv.mongoFlwEECo.UpdateOne(
			context.TODO(), &bson.M{"userid": followerid}, 
			&bson.M{"$pull": bson.M{"edges": followeeid}})
	}
	if err1 != nil || err2 != nil {
		return res, fmt.Errorf("error updating graph %v %v", err1, err2)
	}
	res.Ok = GRAPH_QUERY_OK
	gsrv.clearCache(ctx, followerid, followeeid)
	return res, nil
}

func (gsrv *GraphSrv) updateGraphWithUname(
		ctx context.Context, follwerUname, followeeUname string, isFollow bool) (
		*proto.GraphUpdateResponse, error) {
	userReq := &userpb.CheckUserRequest{Usernames: []string{follwerUname, followeeUname}}
	userRes, err := gsrv.userc.CheckUser(ctx, userReq)
	if err != nil {
		return nil, err
	} else if userRes.Ok != user.USER_QUERY_OK {
		log.Error().Msgf("Missing user id for %v %v: %v", follwerUname, followeeUname, userRes)
		return &proto.GraphUpdateResponse{Ok: "Follower or Followee does not exist"}, nil
	}
	followerid, followeeid := userRes.Userids[0], userRes.Userids[1]
	return gsrv.updateGraph(ctx, followerid, followeeid, isFollow)
}

func (gsrv *GraphSrv) clearCache(ctx context.Context, followerid, followeeid int64) {
	follower_key := FOLLOWER_CACHE_PREFIX + strconv.FormatInt(followeeid, 10)
	followee_key := FOLLOWEE_CACHE_PREFIX + strconv.FormatInt(followerid, 10)
	if !gsrv.cachec.Delete(ctx, follower_key) {
		log.Error().Msgf("cannot delete followers of %v", follower_key)
	}
	if !gsrv.cachec.Delete(ctx, followee_key) {
		log.Error().Msgf("cannot delete followees of %v", follower_key)
	}
}

// Define getFollowers and getFollowees explicitly for clarity
func (gsrv *GraphSrv) getFollowers(ctx context.Context, userid int64) ([]int64, error) {
	key := FOLLOWER_CACHE_PREFIX + strconv.FormatInt(userid, 10)
	flwERInfo := &EdgeInfo{}
	if followerItem, err := gsrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Debug().Msgf("FollowER %v cache miss", key)
		err = gsrv.mongoFlwERCo.FindOne(
			context.TODO(), &bson.M{"userid": userid}).Decode(&flwERInfo)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return make([]int64, 0), nil
			}
			return nil, err
		}
		log.Debug().Msgf("Found followERs for  %v in DB: %v", userid, flwERInfo)
		encodedFlwERInfo, err := json.Marshal(flwERInfo)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		gsrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedFlwERInfo})
	} else {
		log.Debug().Msgf("Found followERs for %v in cache!", userid)
		json.Unmarshal(followerItem.Value, flwERInfo)
	}	
	return flwERInfo.Edges, nil
}

func (gsrv *GraphSrv) getFollowees(ctx context.Context, userid int64) ([]int64, error) {
	key := FOLLOWEE_CACHE_PREFIX + strconv.FormatInt(userid, 10)
	flwEEInfo := &EdgeInfo{}
	if followeeItem, err := gsrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return nil, err
		}
		log.Debug().Msgf("FollowEE %v cache miss", key)
		err = gsrv.mongoFlwEECo.FindOne(
			context.TODO(), &bson.M{"userid": userid}).Decode(&flwEEInfo)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return make([]int64, 0), nil
			}
			return nil, err
		}
		log.Debug().Msgf("Found followEEs for  %v in DB: %v", userid, flwEEInfo)
		encodedFlwEEInfo, err := json.Marshal(flwEEInfo)	
		if err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		gsrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedFlwEEInfo})
	} else {
		log.Debug().Msgf("Found followEEs for %v in cache!", userid)
		json.Unmarshal(followeeItem.Value, flwEEInfo)
	}	
	return flwEEInfo.Edges, nil
}

type EdgeInfo struct {
	Userid int64   `bson:userid`
	Edges  []int64 `bson:edges`
}
