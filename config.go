package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	ghttp "google.golang.org/api/transport/http"
)

// Config represents the protodash configuration that is loaded from the
// environment.
type Config struct {
	Listen          string        `default:":8080"`
	LogLevel        string        `default:"debug"`
	ProxyTimeout    time.Duration `default:"10s"`
	ClientTimeout   time.Duration `default:"2s"`
	IdleConnTimeout time.Duration `default:"120s"`
	MaxIdleConns    int           `default:"10"`
}

func (c *Config) Logger() *logrus.Logger {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		level = logrus.DebugLevel
	}

	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Level = level
	return logger
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
