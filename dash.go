package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/hlog"
)

// Dash is an instance of a specific dashboard
type Dash struct {
	Name      string
	Slug      string
	Bucket    string `yaml:"gcs_bucket"`
	SPA       bool   `yaml:"single_page_app"`
	Prefix    string
	Public    bool
	Subdomain bool
	Config    *Config
	Client    *http.Client
}

const gcsHost = "storage.googleapis.com"

func (d *Dash) getObject(ctx context.Context, headers http.Header, method, key string) (*http.Response, error) {
	// build up the GCS URL
	url := fmt.Sprintf("https://%s.%s/%s", d.Bucket, gcsHost, key)

	// create the request against GCS
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = headers

	// run the request
	return d.Client.Do(req)

}

func (d *Dash) Handler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// create a timeout on the proxy request
		ctx, cancel := context.WithTimeout(r.Context(), d.Config.ProxyTimeout)
		defer cancel()

		// build the object name
		objName := strings.TrimPrefix(r.URL.Path, prefix)
		if objName == "" || strings.HasSuffix(objName, "/") {
			objName += "index.html"
		}
		if d.Prefix != "" {
			objName = d.Prefix + "/" + objName
		}

		// get the object
		gcsResp, err := d.getObject(ctx, r.Header, r.Method, objName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if gcsResp.StatusCode == http.StatusNotFound {
			if d.SPA && objName != (d.Prefix+"/index.html") {
				objName = "index.html"
				if d.Prefix != "" {
					objName = d.Prefix + "/" + objName
				}
				gcsResp.Body.Close()

				gcsResp, err = d.getObject(ctx, r.Header, r.Method, objName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}

		defer gcsResp.Body.Close()

		// add dashboard name, bucket, and object to log
		hlog.FromRequest(r).Info().
			Str("dashboard", d.Name).
			Str("bucket", d.Bucket).
			Str("object", objName).
			Msg("")

		// copy GCS response headers and body to our response
		for name, values := range gcsResp.Header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(gcsResp.StatusCode)
		if _, err = io.Copy(w, gcsResp.Body); err != nil {
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		}
	}
}
