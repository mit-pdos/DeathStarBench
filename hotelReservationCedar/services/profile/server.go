package profile

import (
	"encoding/json"
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	// "io/ioutil"
	"net"
	// "os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	cacheclnt "github.com/harlow/go-micro-services/cacheclnt"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/profile/proto"
	"github.com/harlow/go-micro-services/tls"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/bradfitz/gomemcache/memcache"
	// "strings"
)

type Hotel struct {
	Id          string   `bson:"id"`
	Name        string   `bson:"name"`
	PhoneNumber string   `bson:"phoneNumber"`
	Description string   `bson:"description"`
	Address     *Address `bson:"address"`
}

type Address struct {
	StreetNumber string  `bson:"streetNumber"`
	StreetName   string  `bson:"streetName"`
	City         string  `bson:"city"`
	State        string  `bson:"state"`
	Country      string  `bson:"country"`
	PostalCode   string  `bson:"postalCode"`
	Lat          float32 `bson:"lat"`
	Lon          float32 `bson:"lon"`
}

const name = "srv-profile"

// Server implements the profile service
type Server struct {
	pb.UnimplementedProfileServer
	cc *cacheclnt.CacheClnt

	Tracer       opentracing.Tracer
	uuid         string
	Port         int
	IpAddr       string
	MongoSession *mgo.Session
	Registry     *registry.Client
	MemcClient   *memcache.Client
}

// Run starts the server
func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	zerolog.SetGlobalLevel(zerolog.PanicLevel)
	//	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	s.uuid = uuid.New().String()
	s.MemcClient.MaxIdleConns = 8000
	s.cc = cacheclnt.MakeCacheClnt()

	log.Trace().Msgf("in run s.IpAddr = %s, port = %d", s.IpAddr, s.Port)

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

	pb.RegisterProfileServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatal().Msgf("failed to configure listener: %v", err)
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

// GetProfiles returns hotel profiles for requested IDs
func (s *Server) GetProfiles(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	// session, err := mgo.Dial("mongodb-profile")
	// if err != nil {
	// 	panic(err)
	// }
	// defer session.Close()

	log.Trace().Msgf("In GetProfiles")

	res := new(pb.Result)
	hotels := make([]*pb.Hotel, 0)

	// one hotel should only have one profile

	for _, i := range req.HotelIds {
		// first check memcached
		var item *memcache.Item
		var err error
		if !cacheclnt.UseCached() {
			item, err = s.MemcClient.Get(i + "-prof")
		} else {
			item, err = s.cc.Get(ctx, i+"-prof")
		}

		if err == nil {
			// memcached hit
			profile_str := string(item.Value)
			log.Trace().Msgf("memc hit with %v", profile_str)

			hotel_prof := new(Hotel)
			err := json.Unmarshal(item.Value, hotel_prof)
			if err != nil {
				log.Panic().Msgf("Error unmarshal mc: %v", err)
			}
			hotels = append(hotels, &pb.Hotel{
				Id:          hotel_prof.Id,
				Name:        hotel_prof.Name,
				PhoneNumber: hotel_prof.PhoneNumber,
				Description: hotel_prof.Description,
				Address: &pb.Address{
					StreetNumber: hotel_prof.Address.StreetNumber,
					StreetName:   hotel_prof.Address.StreetName,
					City:         hotel_prof.Address.City,
					State:        hotel_prof.Address.State,
					Country:      hotel_prof.Address.Country,
					PostalCode:   hotel_prof.Address.PostalCode,
					Lat:          hotel_prof.Address.Lat,
					Lon:          hotel_prof.Address.Lon,
				},
			})

		} else if err == memcache.ErrCacheMiss {
			// memcached miss, set up mongo connection
			session := s.MongoSession.Copy()
			defer session.Close()
			c := session.DB("profile-db").C("hotels")

			hotel_prof := new(Hotel)
			err := c.Find(bson.M{"id": i}).One(&hotel_prof)

			if err != nil {
				log.Error().Msgf("Failed get hotels data: ", err)
			}

			// for _, h := range hotels {
			// 	res.Hotels = append(res.Hotels, h)
			// }
			hotels = append(hotels, &pb.Hotel{
				Id:          hotel_prof.Id,
				Name:        hotel_prof.Name,
				PhoneNumber: hotel_prof.PhoneNumber,
				Description: hotel_prof.Description,
				Address: &pb.Address{
					StreetNumber: hotel_prof.Address.StreetNumber,
					StreetName:   hotel_prof.Address.StreetName,
					City:         hotel_prof.Address.City,
					State:        hotel_prof.Address.State,
					Country:      hotel_prof.Address.Country,
					PostalCode:   hotel_prof.Address.PostalCode,
					Lat:          hotel_prof.Address.Lat,
					Lon:          hotel_prof.Address.Lon,
				},
			})

			prof_json, err := json.Marshal(hotel_prof)
			if err != nil {
				log.Error().Msgf("Failed to marshal hotel [id: %v] with err:", hotel_prof.Id, err)
			}
			memc_str := string(prof_json)

			// write to memcached
			item := &memcache.Item{Key: i + "-prof", Value: []byte(memc_str)}
			if !cacheclnt.UseCached() {
				s.MemcClient.Set(item)
			} else {
				s.cc.Set(ctx, item)
			}

		} else {
			log.Panic().Msgf("Tried to get hotelId [%v], but got memmcached error = %s", i, err)
		}
	}

	res.Hotels = hotels
	log.Trace().Msgf("In GetProfiles after getting resp")
	return res, nil
}
