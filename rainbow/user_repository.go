package rainbow

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"bitbucket.org/morrocio/kafra/util"
)

// UserRepository ...
type UserRepository interface {
	Create(username string, password string) (string, error)
	Login(username string, password string) (string, error)
	Validate(userID string) bool
}

type userEntity struct {
	id       string
	username string
	password string
	salt     string
}

// InMemoryUserRepository ...
type InMemoryUserRepository struct {
	users map[string]*userEntity
}

// NewInMemoryUserRepository ... is a default in-memory user repository used for quick development
func NewInMemoryUserRepository() UserRepository {
	return &InMemoryUserRepository{users: make(map[string]*userEntity, 0)}
}

// Create ... register a new user
func (r *InMemoryUserRepository) Create(username string, password string) (string, error) {
	salt := randomSalt()
	pw := util.HashSHA256(password + salt)

	e := userEntity{
		id:       randomID(),
		username: username,
		password: pw,
		salt:     salt,
	}

	r.users[username] = &e
	return e.id, nil
}

// Login ...
func (r *InMemoryUserRepository) Login(username string, password string) (string, error) {
	e := r.users[username]
	if e == nil {
		return "", fmt.Errorf("invalid username")
	}

	pw := util.HashSHA256(password + e.salt)
	if e.password != pw {
		return "", fmt.Errorf("invalid password")
	}

	return e.id, nil
}

// Validate ...
func (r *InMemoryUserRepository) Validate(userID string) bool {
	for _, e := range r.users {
		if e.id == userID {
			return true
		}
	}
	return false
}

func randomID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func randomSalt() string {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}
