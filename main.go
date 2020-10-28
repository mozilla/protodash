package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/gobuffalo/flect"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/mozilla/protodash/pkce"
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
	public := newLoggingChain()
	private := public

	// configure authentication if enabled
	if cfg.OAuthEnabled {
		cookieStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))
		cookieStore.Options.HttpOnly = true
		gothic.Store = cookieStore

		goth.UseProviders(pkce.New(
			cfg.OAuthClientID,
			cfg.OAuthRedirectURI,
			cfg.OAuthDomain,
		))

		gothic.GetProviderName = func(req *http.Request) (string, error) {
			return "pkce", nil
		}

		log.Info().Msg("enabling oauth authentication")

		http.Handle("/auth/login", public.ThenFunc(login))
		http.Handle("/auth/callback", public.ThenFunc(callback))
		http.Handle("/auth/logout", public.ThenFunc(logout))

		private = public.Append(requireAuth)
	}

	// mount the index function to "/"
	http.Handle("/", public.ThenFunc(index(dashboards, tmpl, cfg.OAuthEnabled)))

	// iterate over the dashboards and mount them
	for _, dashboard := range dashboards {
		log.Info().Msgf("mounting %s at /%s/", dashboard.Name, dashboard.Slug)
		chain := private
		if dashboard.Public {
			chain = public
		}
		http.Handle("/"+dashboard.Slug+"/", chain.Then(dashboard))
	}

	if err = http.ListenAndServe(cfg.Listen, nil); err != nil {
		log.Fatal().Err(err).Send()
	}
}

type indexData struct {
	Dashboards  []*Dash
	AuthEnabled bool
	UserEmail   string
}

func index(dashboards []*Dash, tmpl *template.Template, authEnabled bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return 404 if not the root
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := &indexData{
			Dashboards:  dashboards,
			AuthEnabled: authEnabled,
		}

		if authEnabled {
			session, _ := gothic.Store.Get(r, sessionName)
			if email, ok := session.Values["current_user_email"]; ok {
				data.UserEmail = email.(string)
			}
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Error().Err(err).Send()
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		}
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
