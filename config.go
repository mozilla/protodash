package main

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/option"
	ghttp "google.golang.org/api/transport/http"
)

// Config represents the protodash configuration that is loaded from the
// environment.
type Config struct {
	Listen          string        `default:":8080"`
	LogLevel        string        `split_words:"true" default:"debug"`
	ProxyTimeout    time.Duration `split_words:"true" default:"10s"`
	ClientTimeout   time.Duration `split_words:"true" default:"2s"`
	IdleConnTimeout time.Duration `split_words:"true" default:"120s"`
	MaxIdleConns    int           `split_words:"true" default:"10"`

	OAuthEnabled     bool   `envconfig:"OAUTH_ENABLED"`
	OAuthDomain      string `envconfig:"OAUTH_DOMAIN"`
	OAuthClientID    string `envconfig:"OAUTH_CLIENT_ID"`
	OAuthRedirectURI string `envconfig:"OAUTH_REDIRECT_URI"`
	SessionSecret    string `split_words:"true"`
}

// HTTPClient returns an HTTP client with the proper authentication config
// (using Google's default application credentials) and timeouts.
func (c *Config) HTTPClient() (*http.Client, error) {
	baseTransport := &http.Transport{
		IdleConnTimeout: c.IdleConnTimeout,
		MaxIdleConns:    c.MaxIdleConns,
	}
	transport, err := ghttp.NewTransport(
		context.Background(),
		baseTransport,
		option.WithScopes(storage.ScopeReadOnly),
	)
	return &http.Client{
		Timeout:   c.ClientTimeout,
		Transport: transport,
	}, err
}

// LoadConfig loads the configuration from environment variables.
func LoadConfig() (*Config, error) {
	c := &Config{}
	err := envconfig.Process("protodash", c)
	return c, err
}
