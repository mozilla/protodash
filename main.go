package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gobuffalo/flect"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	dashboards, err := loadDashboards("config.yml", cfg)
	if err != nil {
		log.Fatalln(err)
	}

	tmpl, err := template.ParseFiles("index.gohtml")
	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := struct{ Dashboards []*Dash }{dashboards}
		tmpl.Execute(w, data)
	})

	for _, dashboard := range dashboards {
		http.Handle("/"+dashboard.Slug+"/", dashboard)
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
		dashboard.Logger = config.Logger()
		dashboard.Client, err = config.HTTPClient()
		if err != nil {
			return nil, err
		}
		dashboards = append(dashboards, dashboard)
	}

	return dashboards, nil
}
