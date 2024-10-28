package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/harlow/go-micro-services/registry"
	"github.com/harlow/go-micro-services/services/geo"
	"github.com/harlow/go-micro-services/tracing"
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

	log.Info().Msgf("Read database URL: %v", result["GeoMongoAddress"])
	log.Info().Msg("Initializing DB connection...")
	mongo_session := initializeDatabase(result["GeoMongoAddress"])
	defer mongo_session.Close()
	log.Info().Msg("Successfull")

	serv_port, _ := strconv.Atoi(result["GeoPort"])
	serv_ip := os.Getenv("POD_IP_ADDR")
	if serv_ip == "" {
		log.Fatal().Msgf("No POD_IP_ADDR supplied")
	}

	n_index_str := os.Getenv("N_INDEX")
	if n_index_str == "" {
		log.Fatal().Msgf("No N_INDEX supplied")
	}
	n_index, err := strconv.Atoi(n_index_str)
	if err != nil {
		log.Fatal().Msgf("Can't convert nindex: %v", err)
	}
	searchRadiusStr := os.Getenv("SEARCH_RADIUS")
	if searchRadiusStr == "" {
		log.Fatal().Msgf("No search radius supplied")
	}
	searchRadius, err := strconv.Atoi(searchRadiusStr)
	if err != nil {
		log.Fatal().Msgf("Can't convert searchRadius: %v", err)
	}
	nResultsStr := os.Getenv("N_RESULTS")
	if nResultsStr == "" {
		log.Fatal().Msgf("No search radius supplied")
	}
	nResults, err := strconv.Atoi(nResultsStr)
	if err != nil {
		log.Fatal().Msgf("Can't convert searchRadius: %v", err)
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

	log.Info().Msgf("Initializing jaeger agent [service name: %v | host: %v]...", "geo", *jaegeraddr)

	tracer, err := tracing.Init("geo", *jaegeraddr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing jaeger agent: %v", err)
	}
	log.Info().Msg("Jaeger agent initialized")

	log.Info().Msgf("Initializing consul agent [host: %v]...", *consuladdr)
	registry, err := registry.NewClient(*consuladdr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing consul agent: %v", err)
	}
	log.Info().Msg("Consul agent initialized")

	srv := &geo.Server{
		// Port:     *port,
		Port:         serv_port,
		IpAddr:       serv_ip,
		Tracer:       tracer,
		Registry:     registry,
		MongoSession: mongo_session,
		NIndex:       n_index,
		SearchRadius: searchRadius,
		NResults:     nResults,
	}

	log.Info().Msg("Starting server...")
	log.Fatal().Msg(srv.Run().Error())
}
