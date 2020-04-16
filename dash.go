package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// Dash is an instance of a specific dashboard
type Dash struct {
	Name   string
	Slug   string
	Bucket string `yaml:"gcs_bucket"`
	SPA    bool   `yaml:"single_page_app"`
	Prefix string
	Config *Config
	Logger *logrus.Logger
	Client *http.Client
}

const gcsHost = "storage.googleapis.com"

func (d *Dash) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// we only except GET and HEAD requests
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// create a timeout on the proxy request
	ctx, cancel := context.WithTimeout(r.Context(), d.Config.ProxyTimeout)
	defer cancel()

	// build the object name
	objName := strings.TrimPrefix(r.URL.Path, "/"+d.Slug+"/")
	if d.Prefix != "" {
		objName = d.Prefix + "/" + objName
	}
	if objName == "" {
		objName = "index.html"
	}

	// build up the GCS URL
	url := fmt.Sprintf("https://%s.%s/%s", d.Bucket, gcsHost, objName)

	// create the request against GCS
	gcsReq, err := http.NewRequestWithContext(ctx, r.Method, url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for name, values := range r.Header {
		for _, value := range values {
			gcsReq.Header.Add(name, value)
		}
	}

	// run the request
	gcsResp, err := d.Client.Do(gcsReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer gcsResp.Body.Close()

	if gcsResp.StatusCode == http.StatusNotFound {
		if strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path += "index.html"
			d.ServeHTTP(w, r)
			return
		} else if d.SPA && r.URL.Path != "/"+d.Slug+"/index.html" {
			r.URL.Path = "/" + d.Slug + "/index.html"
			d.ServeHTTP(w, r)
			return
		}
	}

	// copy GCS response headers and body to our response
	for name, values := range gcsResp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(gcsResp.StatusCode)
	io.Copy(w, gcsResp.Body)
}
