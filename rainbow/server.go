package rainbow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
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
	UseCookieAuth(userRepo UserRepository) Server
	RegisterMiddleware(m Middleware) Server
	ServeTemplate(path string, data interface{}, templates ...string) Server
	DefaultMux() *http.ServeMux
}

type server struct {
	http.Server
	handler   *handler
	redisPool *redis.Pool
	db        *sql.DB
	userRepo  UserRepository
}

// NewServer ... return new server
func NewServer(address string) Server {

	mux := http.NewServeMux()

	h := handler{
		middlewares: make([]Middleware, 0),
		mux:         mux,
	}

	return &server{
		Server: http.Server{
			Addr:    address,
			Handler: &h,
		},
		handler: &h,
	}
}

// DefaultMux ...
func (s *server) DefaultMux() *http.ServeMux {
	return s.handler.mux
}

// RegisterMiddleware ...
func (s *server) RegisterMiddleware(m Middleware) Server {
	s.handler.registerMiddleware(m)
	return s
}

// ServeTemplate ...
func (s *server) ServeTemplate(path string, data interface{}, templates ...string) Server {
	s.DefaultMux().HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles(templates...)
		if err != nil {
			fmt.Println(err)
		}
		t.Execute(w, data)
	})
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

func (s *server) UseCookieAuth(userRepo UserRepository) Server {
	s.userRepo = userRepo
	if s.userRepo == nil {
		s.userRepo = NewInMemoryUserRepository()
	}

	authService := middleware.NewAuthService(s.redisPool)

	config := middleware.DefaultConfig(s.redisPool)
	config.ValidateUserFunc = func(userID string) bool {
		if userID == "" {
			log.Printf("[server] Non logged-in session\n")
			return true
		}
		log.Printf("[server] ValidateUserFunc UserID: %s\n", userID)
		return s.userRepo.Validate(userID)
	}

	m := middleware.NewHTTPDefaultMiddleware(config)
	s.handler.middlewares = append(s.handler.middlewares, Middleware(m))

	s.handler.mux.HandleFunc("/api/users/login", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("[server] error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		b := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}

		err = json.Unmarshal(body, &b)
		if err != nil {
			log.Printf("[server] error unmarshaling body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		userID, err := s.userRepo.Login(b.Username, b.Password)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		authService.CreateLoginSession(userID)
		w.Header().Del("Set-Cookie")
		w.Header().Set("X-User-Id", userID)

		log.Printf("[server] login success userID: %s\n", userID)
		// Wait for middleware to write header
		return
	})

	s.handler.mux.HandleFunc("/api/users/register", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("[server] error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		b := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}

		err = json.Unmarshal(body, &b)
		if err != nil {
			log.Printf("[server] error unmarshaling body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		userID, err := s.userRepo.Create(b.Username, b.Password)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("[server] login success userID: %s\n", userID)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "success: "+userID)
	})
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
