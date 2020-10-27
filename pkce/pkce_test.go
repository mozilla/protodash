package pkce_test

import (
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/markbates/goth"
	"github.com/mozilla/protodash/pkce"
	"github.com/stretchr/testify/assert"
)

const (
	pkceDomain      = "pkce.example.com"
	pkceClientID    = "some-client-id"
	pkceRedirectURI = "/callback"
)

func TestNew(t *testing.T) {
	p := provider()
	expectedProfileURL := fmt.Sprintf("https://%s/userinfo", pkceDomain)

	assert.Equal(t, pkceClientID, p.ClientID)
	assert.Equal(t, expectedProfileURL, p.ProfileURL)
	assert.Equal(t, pkceRedirectURI, p.RedirectURL)
}

func TestImplementsProvider(t *testing.T) {
	p := provider()
	assert.Implements(t, (*goth.Provider)(nil), p)
}

func TestBeginAuth(t *testing.T) {
	p := provider()
	expectedAuthURL := fmt.Sprintf("https://%s/authorize", pkceDomain)

	session, err := p.BeginAuth("test_state")
	assert.NoError(t, err)

	s := session.(*pkce.Session)
	assert.Contains(t, s.AuthURL, expectedAuthURL)
}

func TestUnmarshalSession(t *testing.T) {
	p := provider()
	expectedAuthURL := "https://" + pkceDomain + "/oauth/authorize"

	sessionResp := fmt.Sprintf(`{"AuthURL":"%s","AccessToken":"1234567890"}`, expectedAuthURL)
	session, err := p.UnmarshalSession(sessionResp)
	assert.NoError(t, err)

	s := session.(*pkce.Session)
	assert.Equal(t, expectedAuthURL, s.AuthURL)
	assert.Equal(t, "1234567890", s.AccessToken)
}

func TestFetchUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	sampleResp := `{
  		"email_verified": false,
  		"email": "test.account@userinfo.com",
  		"clientID": "q2hnj2iu...",
  		"updated_at": "2016-12-05T15:15:40.545Z",
  		"name": "test.account@userinfo.com",
  		"picture": "https://s.gravatar.com/avatar/dummy.png",
  		"user_id": "auth0|58454...",
  		"nickname": "test.account",
  		"identities": [
  		  {
      			"user_id": "58454...",
      			"provider": "auth0",
      			"connection": "Username-Password-Authentication",
      			"isSocial": false
    		}],
  		"created_at": "2016-12-05T11:16:59.640Z",
  		"sub": "auth0|58454..."
	}`

	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://%s/userinfo", pkceDomain),
		httpmock.NewStringResponder(200, sampleResp),
	)

	p := provider()

	session, _ := p.BeginAuth("test_state")
	s := session.(*pkce.Session)
	s.AccessToken = "token"

	u, err := p.FetchUser(s)
	assert.NoError(t, err)
	assert.Equal(t, "test.account@userinfo.com", u.Email)
	assert.Equal(t, "auth0|58454...", u.UserID)
	assert.Equal(t, "test.account", u.NickName)
	assert.Equal(t, "test.account@userinfo.com", u.Name)
	assert.Equal(t, "token", u.AccessToken)
}

func provider() *pkce.Provider {
	return pkce.New(
		pkceClientID,
		pkceRedirectURI,
		pkceDomain,
	)
}
