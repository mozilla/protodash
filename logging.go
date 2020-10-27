package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

func configureLogging(logLevel string) {
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		err = fmt.Errorf("Unknown Level String: '%s', defaulting to DebugLevel", level)
		log.Warn().Err(err).Msg("")
		log.Warn().Err(err).Msg("")
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
}

func newLoggingChain() alice.Chain {
	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Logger()

	chain := alice.New(
		hlog.NewHandler(logger),
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Msg("")
		}),
		hlog.RemoteAddrHandler("ip"),
		hlog.UserAgentHandler("user_agent"),
		hlog.RefererHandler("referer"),
	)

	return chain
}
