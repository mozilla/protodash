package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/markbates/goth/gothic"
	"github.com/rs/zerolog/log"
)

const sessionName = "_protodash_session"

func (s *Server) authLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt := r.URL.Query().Get("redirect_to")

		if s.config.RedirectToLogin && rt != "" {
			rtu, err := url.Parse(rt)
			if err != nil {
				log.Error().Err(err).Send()
				http.Error(w, "Invalid URL Format", http.StatusInternalServerError)
				return
			}

			if rtu.Host != "" && rtu.Host != s.config.BaseDomain && !strings.HasSuffix(rtu.Host, "."+s.config.BaseDomain) {
				log.Error().Err(fmt.Errorf("invalid hostname %s", rtu.Host))
				http.Error(w, "Invalid Host", http.StatusInternalServerError)
				return
			}

			session, _ := s.sessionStore.Get(r, sessionName)
			session.Values["redirect_to"] = rtu.String()
			if err = session.Save(r, w); err != nil {
				log.Error().Err(err).Send()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		gothic.BeginAuthHandler(w, r)
	}
}

func (s *Server) authCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			log.Error().Err(err).Send()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session, _ := s.sessionStore.New(r, sessionName)
		session.Values["current_user_id"] = user.UserID
		session.Values["current_user_email"] = user.Email

		redirectTo := "//" + s.config.BaseDomain + "/"
		if val, ok := session.Values["redirect_to"]; ok {
			delete(session.Values, "redirect_to")
			redirectTo = val.(string)
		}

		if err = session.Save(r, w); err != nil {
			log.Error().Err(err).Send()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, redirectTo, http.StatusFound)
	}
}

func (s *Server) authLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.sessionStore.Get(r, sessionName)
		session.Options.MaxAge = -1
		session.Values = make(map[interface{}]interface{})
		if err := session.Save(r, w); err != nil {
			log.Error().Err(err).Send()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "//"+s.config.BaseDomain+"/", http.StatusFound)
	}
}

func (s *Server) buildLoginURL(r *http.Request) string {
	rtu := cloneURL(r.URL)
	if rtu.Host == "" && r.Host != s.config.BaseDomain {
		rtu.Host = r.Host
	}

	uv := &url.Values{}
	uv.Add("redirect_to", rtu.String())

	u := &url.URL{
		Host:     s.config.BaseDomain,
		Path:     "/auth/login",
		RawQuery: uv.Encode(),
	}

	return u.String()
}

func (s *Server) isLoggedIn(r *http.Request) bool {
	session, _ := s.sessionStore.Get(r, sessionName)
	_, ok := session.Values["current_user_id"]
	return ok
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isLoggedIn(r) {
			next.ServeHTTP(w, r)
			return
		}

		if s.config.RedirectToLogin {
			http.Redirect(w, r, s.buildLoginURL(r), http.StatusFound)
			return
		}

		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
	})
}
