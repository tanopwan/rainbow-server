package main

import (
	"log"
	"net/http"

	_ "github.com/tanopwan/cookie-authentication/middleware"
	"github.com/tanopwan/rainbow-server/server"
)

func main() {
	log.Println("[main.go] started")
	m1 := server.Middleware(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[log] middleware1 request\n")
			h.ServeHTTP(w, r)
			log.Printf("[log] middleware1 response\n")
		})
	})
	m2 := server.Middleware(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[log] middleware2 request\n")
			h.ServeHTTP(w, r)
			log.Printf("[log] middleware2 response\n")
		})
	})
	server.NewServer(":8081").UseRedis().RegisterMiddleware(m1).RegisterMiddleware(m2).Start()
}
