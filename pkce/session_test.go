package pkce_test

import (
	"testing"

	"github.com/markbates/goth"
	"github.com/mozilla/protodash/pkce"
	"github.com/stretchr/testify/assert"
)

func TestImplementsSession(t *testing.T) {
	s := &pkce.Session{}
	assert.Implements(t, (*goth.Session)(nil), s)
}

func TestGetAuthURL(t *testing.T) {
	s := &pkce.Session{}

	_, err := s.GetAuthURL()
	assert.Error(t, err)

	s.AuthURL = "/foo"
	url, _ := s.GetAuthURL()
	assert.Equal(t, "/foo", url)
}

func TestMarshal(t *testing.T) {
	s := &pkce.Session{}
	data := s.Marshal()
	assert.Equal(t, `{"AuthURL":"","AccessToken":"","RefreshToken":"","ExpiresAt":"0001-01-01T00:00:00Z","CodeVerifier":""}`, data)
}
