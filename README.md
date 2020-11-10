[![CircleCI](https://circleci.com/gh/mozilla/protodash.svg?style=shield&circle-token=742fb1108f7e6e5a28c11d43b21f62605037f5a4)](https://circleci.com/gh/mozilla/protodash)

# ProtoDash

ProtoDash is a tool to aid the rapid development of prototype dashboards and enable data engineering and data science to deploy static sites without the need to engage with ops.

## Dashboard Config

The config for the dashboards is stored in `config.yml` and is a map of slugs (the path that the dashboard will serve from) and the config options for that specific dashboard.

A verbose example of the file with all available options is below.

```yaml
---
dashboard-slug:
  gcs_bucket: my-sandbox-bucket # required
  single_page_app: true # optional
  prefix: sub-dir-in-gcs # optional
  public: true # optional
```

| Key               | Description                                                                                                                 | Default | Required |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------- | ------- | -------- |
| `gcs_bucket`      | The bucket to serve the files from                                                                                          |         | `yes`    |
| `single_page_app` | Whether the app is an SPA, when this is set to `true` and a path would return a 404, we serve the root `index.html` instead | `false` | `no`     |
| `prefix`          | A prefix in the bucket to serve from, this would allow you to run multiple apps from the same bucket                        |         | `no`     |
| `public`          | Whether the dashboard should be publicly accessible                                                                         | `false` | `no`     |

## Adding a dashboard

Protodash has access to GCS buckets created in projects in the `dataops/sandbox` hierarchy: for more information on creating such a project see [Creating a Prototype Data Project on Google Cloud Platform](https://docs.telemetry.mozilla.org/cookbooks/gcp-projects.html).

Once you have a dashboard ready to go, open a PR against `config.yml` with the required info. After it's approved and merged we auto-deploy the changes and from that point on you can edit the files in your GCS bucket and the changes should be instant.

## Local Development

You'll want to set the `GOOGLE_CLOUD_CREDENTIALS` to point at either the json keyfile for a service account, or your local application default credentials json keyfile (usually `~/.config/gcloud/application_default_credentials.json`).

```
go build -o protodash
./protodash
```

## Environment Config

These environment variables control how ProtoDash operates in production. It should not normally be necessary to modify these.

| Env Variable                    | Description                                                                                             | Default |
| ------------------------------- | ------------------------------------------------------------------------------------------------------- | ------- |
| `PROTODASH_LISTEN`              | Address to bind the server                                                                              | `:8080` |
| `PROTODASH_LOG_LEVEL`           | Logging level                                                                                           | `debug` |
| `PROTODASH_PROXY_TIMEOUT`       | Defines the maximum time in serving the proxy requests, this is a hard timeout and includes retries     | `10s`   |
| `PROTODASH_CLIENT_TIMEOUT`      | Hard timeout on requests that protodash sends to the Google Storage API                                 | `2s`    |
| `PROTODASH_IDLE_CONN_TIMEOUT`   | Maximum duration of idle connections between protodash and the Google Storage API                       | `120s`  |
| `PROTODASH_MAX_IDLE_CONNS`      | Maximum number of idle connections to keep open. This doesn't control the maximum number of connections | `10`    |
| `PROTODASH_OAUTH_ENABLED`       | Toggles whether authentication is on or off                                                             | `false` |
| `PROTODASH_OAUTH_DOMAIN`        | The OAuth domain that the authentication layer will use, currently only supports Auth0                  |         |
| `PROTODASH_OAUTH_CLIENT_ID`     | Client ID of the OAuth application                                                                      |         |
| `PROTODASH_OAUTH_CLIENT_SECRET` | Client Secret of the OAuth application, if not defined use the PKCE flow                                |         |
| `PROTODASH_OAUTH_REDIRECT_URI`  | Callback URI to redirect to after authenticating                                                        |         |
| `PROTODASH_SESSION_SECRET`      | Secret to usse for encrypting the session cookie                                                        |         |
| `PROTODASH_SHOW_PRIVATE`        | Whether to show the list of private dashboards if not authenticated                                     | `false` |
| `PROTODASH_REDIRECT_TO_LOGIN`   | Whether to redirect to the login pagee if a user is not authenticated and accesses a private dashboard  | `false` |

## Thanks

- [nytimes/gcs-helper](https://github.com/nytimes/gcs-helper) - Portions of the code here were heavily inspired by the gcs-helper project from the NY Times, particularly the method of proxying requests to GCS without having to use the GCS storage APIs.
- [markbates/goth](https://github.com/markbates/goth) - The core of the authentication / the interfaces we use for the auth are based on those in Goth.
