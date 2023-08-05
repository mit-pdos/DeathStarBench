package registry

import (
	"encoding/json"
	"io/ioutil"
	"flag"
	"strconv"
	"fmt"
	"net"
	"os"
	"socialnetworkk8/tracing"
	consul "github.com/hashicorp/consul/api"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

// NewClient returns a new Client with connection to consul
func NewClient(addr string) (*Client, error) {
	cfg := consul.DefaultConfig()
	cfg.Address = addr

	c, err := consul.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{c}, nil
}

// Client provides an interface for communicating with registry
type Client struct {
	*consul.Client
}

// Look for the network device being dedicated for gRPC traffic.
// The network CDIR should be specified in os environment
// "DSB_HOTELRESERV_GRPC_NETWORK".
// If not found, return the first non loopback IP address.
func getLocalIP() (string, error) {
	var ipGrpc string
	var ips []net.IP

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("registry: can not find local ip")
	} else if len(ips) > 1 {
		// by default, return the first network IP address found.
		ipGrpc = ips[0].String()

		grpcNet := os.Getenv("DSB_GRPC_NETWORK")
		_, ipNetGrpc, err := net.ParseCIDR(grpcNet)
		if err != nil {
			log.Error().Msgf("An invalid network CIDR is set in environment DSB_HOTELRESERV_GRPC_NETWORK: %v", grpcNet)
		} else {
			for _, ip := range ips {
				if ipNetGrpc.Contains(ip) {
					ipGrpc = ip.String()
					log.Info().Msgf("gRPC traffic is routed to the dedicated network %s", ipGrpc)
					break
				}
			}
		}
	} else {
		// only one network device existed
		ipGrpc = ips[0].String()
	}

	return ipGrpc, nil
}

// Register a service with registry
func (c *Client) Register(name string, id string, ip string, port int) error {
	if ip == "" {
		var err error
		ip, err = getLocalIP()
		if err != nil {
			return err
		}
	}
	reg := &consul.AgentServiceRegistration{
		ID:      id,
		Name:    name,
		Port:    port,
		Address: ip,
	}
	log.Info().Msgf("Trying to register service [ name: %s, id: %s, address: %s:%d ]", name, id, ip, port)
	return c.Agent().ServiceRegister(reg)
}

// Deregister removes the service address from registry
func (c *Client) Deregister(id string) error {
	return c.Agent().ServiceDeregister(id)
}

func RegisterByConfig(srv string) (*Client, opentracing.Tracer, string, int, string) {
	log.Info().Msg("Reading config...")
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Error().Msgf("Got error while reading config: %v", err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]string
	json.Unmarshal([]byte(byteValue), &result)

	log.Info().Msg("Successfull")

	serv_port, _ := strconv.Atoi(result[srv + "Port"])
	serv_ip := result[srv + "IP"]
	mongoUrl := "mongodb://" + result["MongoAddress"]
	log.Info().Msgf("Read target port: %v", serv_port)
	log.Info().Msgf("Read consul address: %v", result["consulAddress"])
	log.Info().Msgf("Read jaeger address: %v", result["jaegerAddress"])
	var (
		jaegeraddr = flag.String("jaegeraddr", result["jaegerAddress"], "Jaeger address")
		consuladdr = flag.String("consuladdr", result["consulAddress"], "Consul address")
	)
	flag.Parse()

	log.Info().Msgf("Initializing jaeger [service name: %v | host: %v]...", srv, *jaegeraddr)
	tracer, err := tracing.Init(srv, *jaegeraddr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing jaeger agent: %v", err)
	}
	log.Info().Msg("Jaeger agent initialized")

	log.Info().Msgf("Initializing consul agent [host: %v]...", *consuladdr)
	registry, err := NewClient(*consuladdr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing consul agent: %v", err)
	}
	log.Info().Msg("Consul agent initialized")
	return registry, tracer, serv_ip, serv_port, mongoUrl
}
