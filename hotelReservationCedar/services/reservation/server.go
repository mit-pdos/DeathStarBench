package reservation

import (
	// "encoding/json"
	"fmt"
	log2 "log"
	"net/http"
	"net/http/pprof"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	cacheclnt "github.com/harlow/go-micro-services/cacheclnt"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/reservation/proto"
	"github.com/harlow/go-micro-services/tls"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	// "io/ioutil"
	"net"
	// "os"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	// "strings"
	"strconv"
)

var memcComponentTag = opentracing.Tag{string(ext.Component), "MEMC"}
var dbComponentTag = opentracing.Tag{string(ext.Component), "DB"}

const name = "srv-reservation"

// Server implements the user service
type Server struct {
	pb.UnimplementedReservationServer
	cc *cacheclnt.CacheClnt

	Tracer       opentracing.Tracer
	Port         int
	IpAddr       string
	MongoSession *mgo.Session
	Registry     *registry.Client
	MemcClient   *memcache.Client
	uuid         string
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

	pb.RegisterReservationServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
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

	http.Handle("/pprof/cpu", http.HandlerFunc(pprof.Profile))
	go func() {
		log2.Fatalf("Error ListenAndServe: %v", http.ListenAndServe(":5555", nil))
	}()

	log.Trace().Msgf("In reservation s.IpAddr = %s, port = %d", s.IpAddr, s.Port)

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

// MakeReservation makes a reservation based on given information
func (s *Server) MakeReservation(ctx context.Context, req *pb.Request) (*pb.Result, error) {

	//	var parentCtx opentracing.SpanContext
	//	if parent := opentracing.SpanFromContext(ctx); parent != nil {
	//		parentCtx = parent.Context()
	//	}

	res := new(pb.Result)
	res.HotelId = make([]string, 0)

	// session, err := mgo.Dial("mongodb-reservation")
	// if err != nil {
	// 	panic(err)
	// }
	// defer session.Close()
	session := s.MongoSession.Copy()
	defer session.Close()

	c := session.DB("reservation-db").C("reservation")
	c1 := session.DB("reservation-db").C("number")

	inDate, _ := time.Parse(
		time.RFC3339,
		req.InDate+"T12:00:00+00:00")

	outDate, _ := time.Parse(
		time.RFC3339,
		req.OutDate+"T12:00:00+00:00")
	hotelId := req.HotelId[0]

	indate := inDate.String()[0:10]

	memc_date_num_map := make(map[string]int)

	for inDate.Before(outDate) {
		// check reservations
		count := 0
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]

		// first check memc
		memc_key := hotelId + "_" + inDate.String()[0:10] + "_" + outdate

		//		getspan := s.Tracer.StartSpan(
		//			"memcached/Get",
		//			opentracing.ChildOf(parentCtx),
		//			ext.SpanKindRPCClient,
		//			memcComponentTag,
		//		)

		var item *memcache.Item
		var err error
		if !cacheclnt.UseCached() {
			item, err = s.MemcClient.Get(memc_key)
		} else {
			item, err = s.cc.Get(ctx, memc_key)
		}
		//		getspan.Finish()
		if err == nil {
			// memcached hit
			count, _ = strconv.Atoi(string(item.Value))
			log.Trace().Msgf("memcached hit %s = %d", memc_key, count)
			memc_date_num_map[memc_key] = count + int(req.RoomNumber)

		} else if err == memcache.ErrCacheMiss {
			//			findspan := s.Tracer.StartSpan(
			//				"db/Find",
			//				opentracing.ChildOf(parentCtx),
			//				ext.SpanKindRPCClient,
			//				dbComponentTag,
			//			)
			// memcached miss
			log.Trace().Msgf("memcached miss")
			reserve := make([]reservation, 0)
			err := c.Find(&bson.M{"hotelId": hotelId, "inDate": indate, "outDate": outdate}).All(&reserve)
			//			findspan.Finish()
			if err != nil {
				log2.Printf("Tried to find hotelId [%v] from date [%v] to date [%v], but got error %v", hotelId, indate, outdate, err)
				log.Panic().Msgf("Tried to find hotelId [%v] from date [%v] to date [%v], but got error", hotelId, indate, outdate, err.Error())
			}

			for _, r := range reserve {
				count += r.Number
			}

			memc_date_num_map[memc_key] = count + int(req.RoomNumber)

		} else {
			log2.Printf("Tried to get memc_key [%v], but got memmcached error = %v", memc_key, err)
			log.Panic().Msgf("Tried to get memc_key [%v], but got memmcached error = %s", memc_key, err)
		}

		//		getspan = s.Tracer.StartSpan(
		//			"memcached/Get",
		//			opentracing.ChildOf(parentCtx),
		//			ext.SpanKindRPCClient,
		//			memcComponentTag,
		//		)

		// check capacity
		// check memc capacity
		memc_cap_key := hotelId + "_cap"
		if !cacheclnt.UseCached() {
			item, err = s.MemcClient.Get(memc_cap_key)
		} else {
			item, err = s.cc.Get(ctx, memc_cap_key)
		}

		//		getspan.Finish()
		hotel_cap := 0
		if err == nil {
			// memcached hit
			hotel_cap, _ = strconv.Atoi(string(item.Value))
			log.Trace().Msgf("memcached hit %s = %d", memc_cap_key, hotel_cap)
		} else if err == memcache.ErrCacheMiss {
			// memcached miss
			var num number
			//			findspan := s.Tracer.StartSpan(
			//				"db/Find",
			//				opentracing.ChildOf(parentCtx),
			//				ext.SpanKindRPCClient,
			//				dbComponentTag,
			//			)
			err = c1.Find(&bson.M{"hotelId": hotelId}).One(&num)
			//			findspan.Finish()
			if err != nil {
				log2.Printf("Tried to find hotelId [%v], but got error", hotelId, err.Error())
				log.Panic().Msgf("Tried to find hotelId [%v], but got error", hotelId, err.Error())
			}
			hotel_cap = int(num.Number)

			//			setspan := s.Tracer.StartSpan(
			//				"memcached/Set",
			//				opentracing.ChildOf(parentCtx),
			//				ext.SpanKindRPCClient,
			//				memcComponentTag,
			//			)
			// write to memcache
			item := &memcache.Item{Key: memc_cap_key, Value: []byte(strconv.Itoa(hotel_cap))}
			if !cacheclnt.UseCached() {
				s.MemcClient.Set(item)
			} else {
				s.cc.Set(ctx, item)
			}

			//			setspan.Finish()
		} else {
			log2.Printf("Tried to get memc_cap_key [%v], but got memmcached error = %s", memc_cap_key, err)
			log.Panic().Msgf("Tried to get memc_cap_key [%v], but got memmcached error = %s", memc_cap_key, err)
		}

		if count+int(req.RoomNumber) > hotel_cap {
			return res, nil
		}
		indate = outdate
	}

	// only update reservation number cache after check succeeds
	for key, val := range memc_date_num_map {
		//		setspan := s.Tracer.StartSpan(
		//			"memcached/Set",
		//			opentracing.ChildOf(parentCtx),
		//			ext.SpanKindRPCClient,
		//			memcComponentTag,
		//		)
		item := &memcache.Item{Key: key, Value: []byte(strconv.Itoa(val))}
		if !cacheclnt.UseCached() {
			s.MemcClient.Set(item)
		} else {
			s.cc.Set(ctx, item)
		}
		//		setspan.Finish()
	}

	inDate, _ = time.Parse(
		time.RFC3339,
		req.InDate+"T12:00:00+00:00")

	indate = inDate.String()[0:10]

	for inDate.Before(outDate) {
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]
		//		insertspan := s.Tracer.StartSpan(
		//			"db/Insert",
		//			opentracing.ChildOf(parentCtx),
		//			ext.SpanKindRPCClient,
		//			dbComponentTag,
		//		)
		err := c.Insert(&reservation{
			HotelId:      hotelId,
			CustomerName: req.CustomerName,
			InDate:       indate,
			OutDate:      outdate,
			Number:       int(req.RoomNumber)})
		//		insertspan.Finish()
		if err != nil {
			log.Panic().Msgf("Tried to insert hotel [hotelId %v], but got error", hotelId, err.Error())
		}
		indate = outdate
	}

	res.HotelId = append(res.HotelId, hotelId)

	return res, nil
}

// CheckAvailability checks if given information is available
func (s *Server) CheckAvailability(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	//	var parentCtx opentracing.SpanContext
	//	if parent := opentracing.SpanFromContext(ctx); parent != nil {
	//		parentCtx = parent.Context()
	//	}

	res := new(pb.Result)
	res.HotelId = make([]string, 0)

	// session, err := mgo.Dial("mongodb-reservation")
	// if err != nil {
	// 	panic(err)
	// }
	// defer session.Close()
	session := s.MongoSession.Copy()
	defer session.Close()

	c := session.DB("reservation-db").C("reservation")
	c1 := session.DB("reservation-db").C("number")

	for _, hotelId := range req.HotelId {
		log.Trace().Msgf("reservation check hotel %s", hotelId)
		inDate, _ := time.Parse(
			time.RFC3339,
			req.InDate+"T12:00:00+00:00")

		outDate, _ := time.Parse(
			time.RFC3339,
			req.OutDate+"T12:00:00+00:00")

		indate := inDate.String()[0:10]

		for inDate.Before(outDate) {
			// check reservations
			count := 0
			inDate = inDate.AddDate(0, 0, 1)
			log.Trace().Msgf("reservation check date %s", inDate.String()[0:10])
			outdate := inDate.String()[0:10]

			//			getspan := s.Tracer.StartSpan(
			//				"memcached/Get",
			//				opentracing.ChildOf(parentCtx),
			//				ext.SpanKindRPCClient,
			//				memcComponentTag,
			//			)

			// first check memc
			memc_key := hotelId + "_" + inDate.String()[0:10] + "_" + outdate
			var item *memcache.Item
			var err error
			if !cacheclnt.UseCached() {
				item, err = s.MemcClient.Get(memc_key)
			} else {
				item, err = s.cc.Get(ctx, memc_key)
			}

			//			getspan.Finish()

			if err == nil {
				// memcached hit
				count, _ = strconv.Atoi(string(item.Value))
				log.Trace().Msgf("memcached hit %s = %d", memc_key, count)
			} else if err == memcache.ErrCacheMiss {

				//				findspan := s.Tracer.StartSpan(
				//					"db/Find",
				//					opentracing.ChildOf(parentCtx),
				//					ext.SpanKindRPCClient,
				//					dbComponentTag,
				//				)
				// memcached miss
				reserve := make([]reservation, 0)
				err := c.Find(&bson.M{"hotelId": hotelId, "inDate": indate, "outDate": outdate}).All(&reserve)
				//				findspan.Finish()
				if err != nil {
					log.Panic().Msgf("Tried to find hotelId [%v] from date [%v] to date [%v], but got error", hotelId, indate, outdate, err.Error())
				}
				for _, r := range reserve {
					log.Trace().Msgf("reservation check reservation number = %d", hotelId)
					count += r.Number
				}

				//				setspan := s.Tracer.StartSpan(
				//					"memcached/Get",
				//					opentracing.ChildOf(parentCtx),
				//					ext.SpanKindRPCClient,
				//					memcComponentTag,
				//				)

				// update memcached
				item := &memcache.Item{Key: memc_key, Value: []byte(strconv.Itoa(count))}
				if !cacheclnt.UseCached() {
					s.MemcClient.Set(item)
				} else {
					s.cc.Set(ctx, item)
				}

				//				setspan.Finish()
			} else {
				log.Panic().Msgf("Tried to get memc_key [%v], but got memmcached error = %s", memc_key, err)

			}

			//			getspan = s.Tracer.StartSpan(
			//				"memcached/Get",
			//				opentracing.ChildOf(parentCtx),
			//				ext.SpanKindRPCClient,
			//				memcComponentTag,
			//			)

			// check capacity
			// check memc capacity
			memc_cap_key := hotelId + "_cap"
			if !cacheclnt.UseCached() {
				item, err = s.MemcClient.Get(memc_cap_key)
			} else {
				item, err = s.cc.Get(ctx, memc_cap_key)
			}

			hotel_cap := 0

			//			getspan.Finish()

			if err == nil {
				// memcached hit
				hotel_cap, _ = strconv.Atoi(string(item.Value))
				log.Trace().Msgf("memcached hit %s = %d", memc_cap_key, hotel_cap)
			} else if err == memcache.ErrCacheMiss {
				var num number
				//				findspan := s.Tracer.StartSpan(
				//					"db/Find",
				//					opentracing.ChildOf(parentCtx),
				//					ext.SpanKindRPCClient,
				//					dbComponentTag,
				//				)
				err = c1.Find(&bson.M{"hotelId": hotelId}).One(&num)
				//				findspan.Finish()
				if err != nil {
					log.Panic().Msgf("Tried to find hotelId [%v], but got error", hotelId, err.Error())
				}
				hotel_cap = int(num.Number)

				//				setspan := s.Tracer.StartSpan(
				//					"memcached/Get",
				//					opentracing.ChildOf(parentCtx),
				//					ext.SpanKindRPCClient,
				//					memcComponentTag,
				//				)
				item := &memcache.Item{Key: memc_cap_key, Value: []byte(strconv.Itoa(hotel_cap))}
				if !cacheclnt.UseCached() {
					s.MemcClient.Set(item)
				} else {
					s.cc.Set(ctx, item)
				}
				// update memcached
				//				setspan.Finish()
			} else {
				log.Panic().Msgf("Tried to get memc_key [%v], but got memmcached error = %s", memc_cap_key, err)
			}

			if count+int(req.RoomNumber) > hotel_cap {
				break
			}
			indate = outdate

			if inDate.Equal(outDate) {
				res.HotelId = append(res.HotelId, hotelId)
			}
		}
	}

	return res, nil
}

type reservation struct {
	HotelId      string `bson:"hotelId"`
	CustomerName string `bson:"customerName"`
	InDate       string `bson:"inDate"`
	OutDate      string `bson:"outDate"`
	Number       int    `bson:"number"`
}

type number struct {
	HotelId string `bson:"hotelId"`
	Number  int    `bson:"numberOfRoom"`
}
