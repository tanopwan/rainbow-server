package server

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/tanopwan/cookie-authentication/middleware"
)

// Server ...
type Server interface {
	Start()
	UseRedis() Server
	RegisterMiddleware(m Middleware) Server
}

type server struct {
	http.Server
	handler   *handler
	redisPool *redis.Pool
	db        *sql.DB
}

// NewServer ... return new server
func NewServer(address string) Server {

	h := handler{
		middlewares: make([]Middleware, 0),
	}

	return &server{
		Server: http.Server{
			Addr:    address,
			Handler: &h,
		},
		handler: &h,
	}
}

// RegisterMiddleware ...
func (s *server) RegisterMiddleware(m Middleware) Server {
	s.handler.registerMiddleware(m)
	return s
}

// UseRedis ...
func (s *server) UseRedis() Server {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	log.Println("[server] connecting to redis pool at", redisHost)
	redisPool := &redis.Pool{
		MaxIdle:     2,
		IdleTimeout: 60 * time.Minute,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", redisHost+":"+redisPort)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	s.redisPool = redisPool

	return s
}

func (s *server) UseCookieAuth() Server {
	config := middleware.DefaultConfig(s.redisPool)
	config.ValidateUserFunc = func(userID string) bool {
		if userID == "" {
			log.Printf("[server] Non logged-in session\n")
			return true
		}
		log.Printf("[server] ValidateUserFunc UserID: %s\n", userID)
		return true
	}

	m := middleware.NewHTTPDefaultMiddleware(config)
	s.handler.middlewares = append(s.handler.middlewares, Middleware(m))
	return s
}

func (s *server) Start() {
	go func() {
		log.Printf("[server] starting server at %s\n", s.Server.Addr)
		err := s.ListenAndServe()
		if err != http.ErrServerClosed {
			panic(err)
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGTERM)

	<-stop

	log.Println("[server] shutting down ... SIGTERM received")
	// pkill -15 main
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err := s.Shutdown(ctx)
	if err != nil {
		panic(err)
	}
}

// Middleware ...
type Middleware func(http.Handler) http.Handler

type handler struct {
	middlewares []Middleware
}

func (a *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handler\n")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello world\n")
	})

	ms := http.Handler(h)
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		m := a.middlewares[i]
		ms = m(ms)
	}

	ms.ServeHTTP(w, r)
}

func (a *handler) registerMiddleware(m Middleware) {
	a.middlewares = append(a.middlewares, m)
}