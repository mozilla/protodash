package main

import "github.com/gorilla/sessions"

type Server struct {
	config       *Config
	sessionStore sessions.Store
}
