package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/gobuffalo/flect"
	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Panic().Err(err).Send()
	}

	dashboards, err := loadDashboards("config.yml", cfg)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	tmpl, err := template.ParseFiles("index.gohtml")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		err = fmt.Errorf("Unknown Level String: '%s', defaulting to DebugLevel", level)
		log.Warn().Err(err).Msg("")
		log.Warn().Err(err).Msg("")
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Logger()

	chain := alice.New(hlog.NewHandler(logger))

	chain = chain.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	chain = chain.Append(hlog.RemoteAddrHandler("ip"))
	chain = chain.Append(hlog.UserAgentHandler("user_agent"))
	chain = chain.Append(hlog.RefererHandler("referer"))

	http.Handle("/", chain.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := struct{ Dashboards []*Dash }{dashboards}
		tmpl.Execute(w, data)
	}))

	for _, dashboard := range dashboards {
		log.Info().Msgf("mounting %s at /%s/", dashboard.Name, dashboard.Slug)
		http.Handle("/"+dashboard.Slug+"/", chain.Then(dashboard))
	}

	http.ListenAndServe(":8080", nil)
}

func loadDashboards(name string, config *Config) ([]*Dash, error) {
	cfgFile, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	var dashboardMap map[string]*Dash
	d := yaml.NewDecoder(cfgFile)
	if err := d.Decode(&dashboardMap); err != nil {
		return nil, err
	}

	var dashboards []*Dash
	for slug, dashboard := range dashboardMap {
		dashboard.Slug = slug
		dashboard.Name = flect.Titleize(slug)
		dashboard.Config = config
		dashboard.Client, err = config.HTTPClient()
		if err != nil {
			return nil, err
		}
		dashboards = append(dashboards, dashboard)
	}

	return dashboards, nil
}
