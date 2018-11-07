package jrpc2

import (
	"net/http"
)

// Create creates a new service instance
func Create(host, route string, headers map[string]string) *Service {
	return &Service{
		Host:    host,
		Route:   route,
		Methods: make(map[string]Method),
		Headers: headers,
	}
}

// Register maps the provided method to the given name for later method calls.
func (s *Service) Register(name string, method Method) {
	s.Methods[name] = method
}

// Start binds the RPCHandler to the server route and starts the http server
func (s *Service) Start() {
	http.HandleFunc(s.Route, s.RPCHandler)

	err := http.ListenAndServe(s.Host, nil)
	if err != nil {
		panic(err.Error())
	}
}
