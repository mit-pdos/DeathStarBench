package url

import (
	"encoding/json"
	"fmt"
	"strings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net"
	"github.com/google/uuid"
	"socialnetworkk8/registry"
	"socialnetworkk8/tune"
	"socialnetworkk8/services/cacheclnt"
	"socialnetworkk8/tls"
	"socialnetworkk8/services/url/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	URL_SRV_NAME = "srv-url"
	URL_QUERY_OK = "OK"
	URL_CACHE_PREFIX = "url_"
	URL_HOSTNAME = "http://short-url/"
	URL_LENGTH = 10
)

var urlPrefixL = len(URL_HOSTNAME)

type UrlSrv struct {
	proto.UnimplementedUrlServer 
	uuid         string
	cachec       *cacheclnt.CacheClnt
	mongoCo      *mongo.Collection
	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
}

func MakeUrlSrv() *UrlSrv {
	tune.Init()
	registry, tracer, serv_ip, serv_port, mongoUrl := registry.RegisterByConfig("Url")
	cachec := cacheclnt.MakeCacheClnt() 
	mongoClient, err := mongo.Connect(
		context.Background(), options.Client().ApplyURI(mongoUrl).SetMaxPoolSize(2048))
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	collection := mongoClient.Database("socialnetwork").Collection("url")
	indexModel := mongo.IndexModel{Keys: bson.D{{"shorturl", 1}}}
	collection.Indexes().CreateOne(context.TODO(), indexModel)
	return &UrlSrv{Port:serv_port, IpAddr:serv_ip, Tracer:tracer, Registry:registry, cachec:cachec,	mongoCo:collection}
}

// Run starts the server
func (urlsrv *UrlSrv) Run() error {
	if urlsrv.Port == 0 {
		return fmt.Errorf("server port must be set")
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	urlsrv.uuid = uuid.New().String()
	grpcSrv := grpc.NewServer(tls.DefaultOpts()...)
	proto.RegisterUrlServer(grpcSrv, urlsrv)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", urlsrv.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	err = urlsrv.Registry.Register(URL_SRV_NAME, urlsrv.uuid, urlsrv.IpAddr, urlsrv.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")
	return grpcSrv.Serve(lis)
}


func (urlsrv *UrlSrv) ComposeUrls(
		ctx context.Context, req *proto.ComposeUrlsRequest) (*proto.ComposeUrlsResponse, error) {
	log.Debug().Msgf("Received compose request %v", req)
	nUrls := len(req.Extendedurls)
	res := &proto.ComposeUrlsResponse{}
	if nUrls == 0 {
		res.Ok = "Empty input"
		return res, nil
	}
	res.Shorturls = make([]string, nUrls)
	for idx, extendedurl := range req.Extendedurls {
		shorturl := RandStringRunes(URL_LENGTH)
		url := &Url{Extendedurl: extendedurl, Shorturl: shorturl}
		if _, err := urlsrv.mongoCo.InsertOne(context.TODO(), url); err != nil {
			log.Error().Msg(err.Error())
			return nil, err
		}
		res.Shorturls[idx] = URL_HOSTNAME + shorturl
	} 
	
	res.Ok = URL_QUERY_OK
	return res, nil
}

func (urlsrv *UrlSrv) GetUrls(
		ctx context.Context, req *proto.GetUrlsRequest) (*proto.GetUrlsResponse, error) {
	log.Debug().Msgf("Received get request %v", req)
	res := &proto.GetUrlsResponse{}
	res.Ok = "No."
	extendedurls := make([]string, len(req.Shorturls))
	missing := false
	for idx, shorturl := range req.Shorturls {
		extendedurl, err := urlsrv.getExtendedUrl(ctx, shorturl)
		if err != nil {
			return nil, err
		} 
		if extendedurl == "" {
			missing = true
			res.Ok = res.Ok + fmt.Sprintf(" Missing %v.", shorturl)
		} else {
			extendedurls[idx] = extendedurl	
		}
	}
	res.Extendedurls = extendedurls
	if !missing {
		res.Ok = URL_QUERY_OK
	}
	return res, nil
}

func (urlsrv *UrlSrv) getExtendedUrl(ctx context.Context, shortUrl string) (string, error) {
	if !strings.HasPrefix(shortUrl, URL_HOSTNAME) {
		log.Warn().Msgf("Url %v does not start with %v!", shortUrl, URL_HOSTNAME)
		return "", nil
	}
	urlKey := shortUrl[urlPrefixL:]
	key := URL_CACHE_PREFIX + urlKey
	url := &Url{}
	if urlItem, err := urlsrv.cachec.Get(ctx, key); err != nil {
		if err != memcache.ErrCacheMiss {
			return "", err
		}
		log.Debug().Msgf("url %v cache miss", key)
		err = urlsrv.mongoCo.FindOne(context.TODO(), &bson.M{"shorturl": urlKey}).Decode(&url)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return "", nil
			}
			return "", err
		} 
		log.Debug().Msgf("Found url %v in DB: %v", shortUrl, url)
		encodedUrl, err := json.Marshal(url)	
		if err != nil {
			log.Error().Msg(err.Error())
			return "", err
		}
		urlsrv.cachec.Set(ctx, &memcache.Item{Key: key, Value: encodedUrl})
	} else {
		log.Debug().Msgf("Found url %v in cache!", key)
		json.Unmarshal(urlItem.Value, url)
	}
	return url.Extendedurl, nil
}


type Url struct {
	Shorturl string    `bson:shorturl`
	Extendedurl string `bson:extendedurl`
}
