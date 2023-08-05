package main

import (
	"os"
	"time"
	"runtime/debug"
	"socialnetworkk8/registry"
	"socialnetworkk8/services/frontend"
	"socialnetworkk8/tune"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	debug.SetGCPercent(-1)
	tune.Init()
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()

	registry, tracer, serv_ip, serv_port, _ := registry.RegisterByConfig("Frontend")
	fsrv := &frontend.FrontendSrv{
		Registry: registry,
		Tracer:   tracer,
		IpAddr:   serv_ip,
		Port:     serv_port,
	}

	log.Info().Msg("Starting server...")
	log.Fatal().Msg(fsrv.Run().Error())
}
