package geo

import (
	// "encoding/json"
	"fmt"
	log2 "log"
	"math/rand"
	"sync"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	// "io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	// "os"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/geo/proto"
	"github.com/harlow/go-micro-services/tls"
	"github.com/mit-pdos/go-geoindex"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	name             = "srv-geo"
	maxSearchRadius  = 10
	maxSearchResults = 5
)

const (
	N_INDEX = 1000
)

type safeIndex struct {
	mu     sync.Mutex
	geoidx *geoindex.ClusteringIndex
}

func makeSafeIndex(session *mgo.Session) *safeIndex {
	return &safeIndex{
		geoidx: newGeoIndex(session),
	}
}

func (si *safeIndex) KNN(center *geoindex.GeoPoint) []geoindex.Point {
	si.mu.Lock()
	defer si.mu.Unlock()

	return si.geoidx.KNearest(
		center,
		maxSearchResults,
		geoindex.Km(maxSearchRadius), func(p geoindex.Point) bool {
			return true
		},
	)
}

// Server implements the geo service
type Server struct {
	pb.UnimplementedGeoServer
	indexes []*safeIndex
	uuid    string

	Registry     *registry.Client
	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	MongoSession *mgo.Session
}

// Run starts the server
func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	zerolog.SetGlobalLevel(zerolog.Disabled)

	if len(s.indexes) == 0 {
		s.indexes = make([]*safeIndex, 0, N_INDEX)
		for i := 0; i < N_INDEX; i++ {
			s.indexes = append(s.indexes, makeSafeIndex(s.MongoSession))
		}
	}

	s.uuid = uuid.New().String()

	// opts := []grpc.ServerOption {
	// 	grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
	// 		PermitWithoutStream: true,
	// 	}),
	// }

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Hour,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(s.Tracer),
		),
	}

	if tlsopt := tls.GetServerOpt(); tlsopt != nil {
		opts = append(opts, tlsopt)
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterGeoServer(srv, s)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// register the service
	// jsonFile, err := os.Open("config.json")
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// defer jsonFile.Close()

	// byteValue, _ := ioutil.ReadAll(jsonFile)

	// var result map[string]string
	// json.Unmarshal([]byte(byteValue), &result)

	// fmt.Printf("geo server ip = %s, port = %d\n", s.IpAddr, s.Port)

	http.Handle("/pprof/cpu", http.HandlerFunc(pprof.Profile))
	go func() {
		log2.Fatalf("Error ListenAndServe: %v", http.ListenAndServe(":5000", nil))
	}()

	err = s.Registry.Register(name, s.uuid, s.IpAddr, s.Port)
	if err != nil {
		return fmt.Errorf("failed register: %v", err)
	}
	log.Info().Msg("Successfully registered in consul")

	return srv.Serve(lis)
}

// Shutdown cleans up any processes
func (s *Server) Shutdown() {
	s.Registry.Deregister(s.uuid)
}

// Nearby returns all hotels within a given distance.
func (s *Server) Nearby(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	log.Trace().Msgf("In geo Nearby")

	var (
		points = s.getNearbyPoints(ctx, float64(req.Lat), float64(req.Lon))
		res    = &pb.Result{}
	)

	log.Trace().Msgf("geo after getNearbyPoints, len = %d", len(points))

	for _, p := range points {
		log.Trace().Msgf("In geo Nearby return hotelId = %s", p.Id())
		res.HotelIds = append(res.HotelIds, p.Id())
	}

	return res, nil
}

func (s *Server) getNearbyPoints(ctx context.Context, lat, lon float64) []geoindex.Point {
	log.Trace().Msgf("In geo getNearbyPoints, lat = %f, lon = %f", lat, lon)

	center := &geoindex.GeoPoint{
		Pid:  "",
		Plat: lat,
		Plon: lon,
	}

	r := rand.Int() % N_INDEX

	return s.indexes[r].KNN(center)
}

// newGeoIndex returns a geo index with points loaded
func newGeoIndex(session *mgo.Session) *geoindex.ClusteringIndex {
	// session, err := mgo.Dial("mongodb-geo")
	// if err != nil {
	// 	panic(err)
	// }
	// defer session.Close()

	log.Trace().Msg("new geo newGeoIndex")

	s := session.Copy()
	defer s.Close()
	c := s.DB("geo-db").C("geo")

	var points []*point
	err := c.Find(bson.M{}).All(&points)
	if err != nil {
		log.Error().Msgf("Failed get geo data: ", err)
	}

	// add points to index
	index := geoindex.NewClusteringIndex()
	for _, point := range points {
		index.Add(point)
	}

	return index
}

type point struct {
	Pid  string  `bson:"hotelId"`
	Plat float64 `bson:"lat"`
	Plon float64 `bson:"lon"`
}

// Implement Point interface
func (p *point) Lat() float64 { return p.Plat }
func (p *point) Lon() float64 { return p.Plon }
func (p *point) Id() string   { return p.Pid }
