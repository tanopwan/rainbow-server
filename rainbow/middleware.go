package rainbow

import (
	"log"
	"net/http"
)

// Middleware ...
type Middleware func(http.Handler) http.Handler

type handler struct {
	middlewares []Middleware
	mux         *http.ServeMux
}

func (a *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mux.ServeHTTP(w, r)
	})

	ms := http.Handler(h)
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		m := a.middlewares[i]
		ms = m(ms)
	}

	log.Println(":rainbow: start request ===>")
	ms.ServeHTTP(w, r)
	log.Println(":rainbow: finish request <===")
}

func (a *handler) registerMiddleware(m Middleware) {
	a.middlewares = append(a.middlewares, m)
}
