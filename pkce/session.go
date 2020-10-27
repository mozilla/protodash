package pkce

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

type Session struct {
	AuthURL      string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	CodeVerifier string
}

// GetAuthURL returns the URL for the authentication end-point for the provider.
func (s *Session) GetAuthURL() (string, error) {
	if s.AuthURL == "" {
		return "", errors.New(goth.NoAuthUrlErrorMessage)
	}
	return s.AuthURL, nil
}

// Marshal generates a string representation of the Session for storing between requests.
func (s *Session) Marshal() string {
	buf, _ := json.Marshal(s)
	return string(buf)
}

// Authorize should validate the data from the provider and return an access token
// that can be stored for later access to the provider.
func (s *Session) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	p := provider.(*Provider)

	token, err := p.Config.Exchange(
		context.Background(),
		params.Get("code"),
		oauth2.SetAuthURLParam("code_verifier", s.CodeVerifier),
	)
	if err != nil {
		return "", err
	}

	if !token.Valid() {
		return "", errors.New("invalid token received from provider")
	}

	s.AccessToken = token.AccessToken
	s.RefreshToken = token.RefreshToken
	s.ExpiresAt = token.Expiry

	return token.AccessToken, nil
}

func (p *Provider) UnmarshalSession(data string) (goth.Session, error) {
	s := &Session{}
	if err := json.Unmarshal([]byte(data), s); err != nil {
		return nil, err
	}
	return s, nil
}
