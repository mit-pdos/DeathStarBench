package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	log2 "log"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/cached"
	"github.com/harlow/go-micro-services/tune"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	tune.Init()
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()

	log.Info().Msg("Reading config...")
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Error().Msgf("Got error while reading config: %v", err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]string
	json.Unmarshal([]byte(byteValue), &result)

	serv_port, _ := strconv.Atoi(result["CachedPort"])
	serv_ip := os.Getenv("POD_IP_ADDR")
	if serv_ip == "" {
		log2.Fatalf("No POD_IP_ADDR supplied")
	}

	log.Info().Msgf("Read target port: %v", serv_port)
	log.Info().Msgf("Read consul address: %v", result["consulAddress"])
	log.Info().Msgf("Read jaeger address: %v", result["jaegerAddress"])
	var (
		// port       = flag.Int("port", 8083, "Server port")
		jaegeraddr = flag.String("jaegeraddr", result["jaegerAddress"], "Jaeger address")
		consuladdr = flag.String("consuladdr", result["consulAddress"], "Consul address")
	)
	flag.Parse()

	log.Info().Msgf("Initializing jaeger agent [service name: %v | host: %v]...", "cached", *jaegeraddr)

	//	tracer, err := tracing.Init("cached", *jaegeraddr)
	//	if err != nil {
	//		log.Panic().Msgf("Got error while initializing jaeger agent: %v", err)
	//	}
	//	log.Info().Msg("Jaeger agent initialized")

	log.Info().Msgf("Initializing consul agent [host: %v]...", *consuladdr)
	registry, err := registry.NewClient(*consuladdr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing consul agent: %v", err)
	}
	log.Info().Msg("Consul agent initialized")

	srv := &cached.Server{
		// Port:     *port,
		Port:     serv_port,
		IpAddr:   serv_ip,
		Tracer:   nil,
		Registry: registry,
	}

	if os.Getenv("NOGC") == "true" {
		debug.SetGCPercent(-1)
	}

	log.Info().Msg("Starting server...")
	log.Fatal().Msg(srv.Run().Error())
}
