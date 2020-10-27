package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/gobuffalo/flect"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

func main() {
	// load config file
	cfg, err := LoadConfig()
	if err != nil {
		log.Panic().Err(err).Send()
	}

	// configure logging
	configureLogging(cfg.LogLevel)

	// load dashboard configs
	dashboards, err := loadDashboards("config.yml", cfg)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// parse index template
	tmpl, err := template.ParseFiles("index.gohtml")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// create chain with http loggin
	chain := newLoggingChain()

	// mount the index function to "/"
	http.Handle("/", chain.ThenFunc(index(dashboards, tmpl)))

	// iterate over the dashboards and mount them
	for _, dashboard := range dashboards {
		log.Info().Msgf("mounting %s at /%s/", dashboard.Name, dashboard.Slug)
		http.Handle("/"+dashboard.Slug+"/", chain.Then(dashboard))
	}

	http.ListenAndServe(cfg.Listen, nil)
}

func index(dashboards []*Dash, tmpl *template.Template) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return 404 if not the root
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := struct{ Dashboards []*Dash }{dashboards}
		tmpl.Execute(w, data)
	})
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
