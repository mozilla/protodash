package main

import (
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/rs/zerolog/log"
)

const sessionName = "_protodash_session"

func login(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}

func callback(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Error().Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	store := map[string]string{
		"current_user_id":    user.UserID,
		"current_user_email": user.Email,
	}

	session, _ := gothic.Store.New(r, sessionName)
	for key, val := range store {
		session.Values[key] = val
	}

	if err = session.Save(r, w); err != nil {
		log.Error().Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, err := gothic.Store.Get(r, sessionName)
	if err != nil {
		log.Error().Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	if err = session.Save(r, w); err != nil {
		log.Error().Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := gothic.Store.Get(r, sessionName)
		if _, ok := session.Values["current_user_id"]; !ok {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
