package pkce

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

const (
	authPath    = "/authorize"
	tokenPath   = "/oauth/token"
	profilePath = "/userinfo"
	protocol    = "https://"
)

type Provider struct {
	*oauth2.Config
	ProfileURL string
	name       string
}

type UserInfo struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	NickName string `json:"nickname"`
	UserID   string `json:"sub"`
}

func New(clientID, redirectURI, domain string, scopes ...string) *Provider {
	p := &Provider{
		Config: &oauth2.Config{
			ClientID:    clientID,
			RedirectURL: redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  protocol + domain + authPath,
				TokenURL: protocol + domain + tokenPath,
			},
		},
		ProfileURL: protocol + domain + profilePath,
		name:       "pkce",
	}
	if len(scopes) > 0 {
		p.Config.Scopes = make([]string, len(scopes))
		for i, scope := range scopes {
			p.Config.Scopes[i] = scope
		}
	} else {
		p.Config.Scopes = []string{"openid", "profile", "email"}
	}
	return p
}

func (p *Provider) Name() string {
	return p.name
}

func (p *Provider) SetName(name string) {
	p.name = name
}

func (p *Provider) BeginAuth(state string) (goth.Session, error) {
	cv, err := codeVerifier()
	if err != nil {
		return nil, err
	}

	cc := codeChallenge(cv)

	s := &Session{
		AuthURL: p.Config.AuthCodeURL(
			state,
			oauth2.SetAuthURLParam("code_challenge", cc),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		),
		CodeVerifier: cv,
	}

	return s, nil
}

func (p *Provider) FetchUser(session goth.Session) (goth.User, error) {
	s := session.(*Session)
	user := goth.User{
		AccessToken:  s.AccessToken,
		Provider:     p.Name(),
		RefreshToken: s.RefreshToken,
		ExpiresAt:    s.ExpiresAt,
	}

	if user.AccessToken == "" {
		return user, fmt.Errorf("%s cannot get user information without accessToken", p.Name())
	}

	req, err := http.NewRequest(http.MethodGet, p.ProfileURL, nil)
	if err != nil {
		return user, err
	}
	req.Header.Set("Authorization", "Bearer "+s.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if resp.Body != nil {
			resp.Body.Close()
		}
		return user, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return user, fmt.Errorf("%s responded with a %d while trying to fetch user information", p.Name(), resp.StatusCode)
	}

	var rawData map[string]interface{}
	userInfo := &UserInfo{}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return user, err
	}

	err = json.Unmarshal(body, &rawData)
	if err != nil {
		return user, err
	}

	err = json.Unmarshal(body, userInfo)
	if err != nil {
		return user, err
	}

	user.Email = userInfo.Email
	user.Name = userInfo.Name
	user.NickName = userInfo.NickName
	user.RawData = rawData
	user.UserID = userInfo.UserID

	return user, nil
}

func (p *Provider) Debug(_ bool) {}

func (p *Provider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	tokenSource := p.Config.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}
	return newToken, nil
}

func (p *Provider) RefreshTokenAvailable() bool {
	return true
}
