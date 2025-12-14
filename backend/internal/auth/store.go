package auth

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type User struct {
	Id           string
	Username     string
	PasswordHash string
}

type UserStore struct {
	users map[string]*User
	mu    sync.RWMutex
}

func NewUserStore() *UserStore {
	return &UserStore{
		users: make(map[string]*User),
	}
}

func (s *UserStore) Create(username, password string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, errors.New("username already exists")
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Id:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
	}
	s.users[username] = user
	slog.Info("successfully created user for username", "username", username)
	return user, nil
}

func (s *UserStore) GetByUsername(username string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("username" + username + " not found")
	}
	return user, nil
}
