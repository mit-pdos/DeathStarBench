package cached

import (
	// "encoding/json"
	"fmt"
	"hash/fnv"
	log2 "log"
	"net/rpc"
	"strconv"
	"sync"

	// "io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	// "os"
	"time"

	"github.com/google/uuid"
	//	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	cacheclnt "github.com/harlow/go-micro-services/cacheclnt"
	"github.com/harlow/go-micro-services/registry"
	pb "github.com/harlow/go-micro-services/services/cached/proto"
	"github.com/harlow/go-micro-services/tls"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	NBIN = 1009
	name = "srv-cached"
)

func key2bin(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	bin := h.Sum32() % NBIN
	return bin
}

type cache struct {
	sync.Mutex
	cache map[string][]byte
}

// Server implements the cached service
type Server struct {
	bins []cache
	shrd string
	uuid string

	Registry *registry.Client
	Tracer   opentracing.Tracer
	Port     int
	IpAddr   string
	pb.UnimplementedCachedServer
}

// Run starts the server
func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	//	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.SetGlobalLevel(zerolog.PanicLevel)

	s.bins = make([]cache, NBIN)
	for i := 0; i < NBIN; i++ {
		s.bins[i].cache = make(map[string][]byte)
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
		grpc.ReadBufferSize(65536),
		grpc.WriteBufferSize(65536),
		grpc.MaxConcurrentStreams(10000),
		//		grpc.UnaryInterceptor(
		//			otgrpc.OpenTracingServerInterceptor(s.Tracer),
		//		),
	}

	if tlsopt := tls.GetServerOpt(); tlsopt != nil {
		opts = append(opts, tlsopt)
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterCachedServer(srv, s)

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
		log2.Fatalf("Error ListenAndServe: %v", http.ListenAndServe(":5555", nil))
	}()

	//	err = s.Registry.Register(name, s.uuid, s.IpAddr, s.Port)
	//	if err != nil {
	//		return fmt.Errorf("failed register: %v", err)
	//	}
	//	log.Info().Msg("Successfully registered in consul")

	// Register this cache with the servers that depend on it.
	s.registerWithServers()

	return srv.Serve(lis)
}

// Shutdown cleans up any processes
func (s *Server) Shutdown() {
	s.Registry.Deregister(s.uuid)
}

func (s *Server) registerWithServers() {
	for _, svc := range []string{"reservation", "rate", "profile"} {
		for {
			c, err := rpc.DialHTTP("tcp", svc+cacheclnt.CACHE_CLNT_PORT)
			if err != nil {
				log2.Printf("Error dial server (%v): %v", svc, err)
				time.Sleep(1 * time.Second)
				continue
			}
			log2.Printf("Success dial server (%v)", svc)
			req := &cacheclnt.RegisterCacheRequest{s.IpAddr + ":" + strconv.Itoa(s.Port)}
			res := &cacheclnt.RegisterCacheResponse{}
			err = c.Call("CacheClnt.RegisterCache", req, res)
			if err != nil {
				log2.Fatalf("Error Call RegisterCache: %v", err)
			}
			break
		}
	}
}

func (s *Server) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResult, error) {
	b := key2bin(req.Key)

	s.bins[b].Lock()
	defer s.bins[b].Unlock()

	s.bins[b].cache[req.Key] = req.Val

	res := &pb.SetResult{}
	res.Ok = true
	return res, nil
}

func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResult, error) {
	st := time.Now()
	res := &pb.GetResult{}

	b := key2bin(req.Key)

	s2 := time.Now()
	s.bins[b].Lock()
	defer s.bins[b].Unlock()
	if time.Since(s2) > 2*time.Millisecond {
		log2.Printf("Long lock acquisition get %v", time.Since(s2))
	}

	res.Val, res.Ok = s.bins[b].cache[req.Key]
	if time.Since(st) > 2*time.Millisecond {
		log2.Printf("Long cache get %v", time.Since(st))
	}
	return res, nil
}
