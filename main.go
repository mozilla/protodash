package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/auth0"
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

	s := &Server{config: cfg}

	// configure logging
	configureLogging(cfg.LogLevel)

	// load dashboard configs
	dashboards, err := loadDashboards(cfg.ConfigFile, cfg)
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

	r := mux.NewRouter()
	r.StrictSlash(true)

	bd := cfg.BaseDomain
	bdr := r.Host(bd).Subrouter()

	// configure authentication if enabled
	if cfg.OAuthEnabled {
		cookieStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))
		cookieStore.Options.HttpOnly = true
		parts := strings.Split(cfg.BaseDomain, ":")
		cookieStore.Options.Domain = parts[0]
		gothic.Store = cookieStore
		s.sessionStore = cookieStore

		pkceProvider := pkce.New(
			cfg.OAuthClientID,
			cfg.OAuthRedirectURI,
			cfg.OAuthDomain,
		)

		auth0Provider := auth0.New(
			cfg.OAuthClientID,
			cfg.OAuthClientSecret,
			cfg.OAuthRedirectURI,
			cfg.OAuthDomain,
			"openid",
			"profile",
			"email",
		)

		goth.UseProviders(
			pkceProvider,
			auth0Provider,
		)

		providerName := auth0Provider.Name()
		if cfg.OAuthClientSecret == "" {
			providerName = pkceProvider.Name()
		}

		gothic.GetProviderName = func(req *http.Request) (string, error) {
			return providerName, nil
		}

		log.Info().Msgf("enabling authentication with %s provider", providerName)

		bdr.Handle("/auth/login", public.Then(s.authLogin())).Methods("GET")
		bdr.Handle("/auth/callback", public.Then(s.authCallback())).Methods("GET")
		bdr.Handle("/auth/logout", public.Then(s.authLogout())).Methods("GET")

		private = public.Append(s.requireAuth)
	}

	// iterate over the dashboards and mount them
	for _, dashboard := range dashboards {
		log.Info().Msgf("mounting %s at /%s/", dashboard.Name, dashboard.Slug)
		chain := private
		if dashboard.Public {
			chain = public
		}

		sd := dashboard.Slug + "." + cfg.BaseDomain
		bdp := "/" + dashboard.Slug + "/"
		sdp := "/"
		sdr := r.Host(sd).Subrouter()

		bdghr := bdr.Methods("GET", "HEAD").Subrouter()
		sdghr := sdr.Methods("GET", "HEAD").Subrouter()

		var bdh http.Handler
		var sdh http.Handler

		if dashboard.Subdomain {
			bdh = chain.Then(redirectToDomain(sd, stripPrefix(bdp)))
			sdh = chain.Then(dashboard.Handler(sdp))
		} else {
			bdh = chain.Then(dashboard.Handler(bdp))
			sdh = chain.Then(redirectToDomain(bd, addPrefix(bdp)))
		}

		bdghr.Handle(bdp, bdh)
		bdghr.PathPrefix(bdp).Handler(bdh)

		if dashboard.Subdomain {
			sdghr.Handle(bdp, bdh)
			sdghr.PathPrefix(bdp).Handler(bdh)
		}

		sdghr.Handle(sdp, sdh)
		sdghr.PathPrefix(sdp).Handler(sdh)
	}

	// mount the index function to "/"
	bdr.Handle("/", public.Then(s.index(dashboards, tmpl))).Methods("GET")

	if err = http.ListenAndServe(cfg.Listen, r); err != nil {
		log.Fatal().Err(err).Send()
	}
}

type modifyPathFn func(path string) string

func redirectToDomain(domain string, fn modifyPathFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newURL := fmt.Sprintf("//%s%s", domain, fn(r.URL.Path))
		http.Redirect(w, r, newURL, http.StatusPermanentRedirect)
	}
}

func addPrefix(prefix string) modifyPathFn {
	return func(path string) string {
		return prefix + strings.TrimPrefix(path, "/")
	}
}

func stripPrefix(prefix string) modifyPathFn {
	return func(path string) string {
		return "/" + strings.TrimPrefix(path, prefix)
	}
}

type indexData struct {
	Dashboards []*Dash
	User       *goth.User
	Config     *Config
}

func (s *Server) index(dashboards []*Dash, tmpl *template.Template) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return 404 if not the root
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := &indexData{
			Dashboards: dashboards,
			Config:     s.config,
		}

		if s.config.OAuthEnabled {
			session, _ := gothic.Store.Get(r, sessionName)
			if email, ok := session.Values["current_user_email"]; ok {
				data.User = &goth.User{
					Email: email.(string),
				}
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
		if dashboard.Bucket == "" {
			dashboard.Bucket = config.DefaultBucket
		}
		dashboard.Config = config
		dashboard.Client, err = config.HTTPClient()
		if err != nil {
			return nil, err
		}
		dashboards = append(dashboards, dashboard)
	}

	return dashboards, nil
}
