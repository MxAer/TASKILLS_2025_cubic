package store

import (
	"database/sql"
	"dice-roller/internal/models"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // Postgres driver
)

type Store struct {
	db *sql.DB
}

func New(dbUrl string) (*Store, error) {
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() {
	s.db.Close()
}

// --- User Methods ---

func (s *Store) CreateUser(username string) (*models.User, error) {
	u := &models.User{Username: username}
	query := `INSERT INTO users (username) VALUES ($1) RETURNING id`
	err := s.db.QueryRow(query, username).Scan(&u.ID)
	if err != nil {
		return nil, fmt.Errorf("create user error: %w", err)
	}
	return u, nil
}

func (s *Store) GetUser(id uuid.UUID) (*models.User, error) {
	u := &models.User{}
	query := `SELECT id, username FROM users WHERE id = $1`
	err := s.db.QueryRow(query, id).Scan(&u.ID, &u.Username)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// --- Room Methods ---

func (s *Store) CreateRoom() (*models.Room, error) {
	r := &models.Room{}
	query := `INSERT INTO rooms (created_at) VALUES (NOW()) RETURNING id, created_at`
	err := s.db.QueryRow(query).Scan(&r.ID, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create room error: %w", err)
	}
	return r, nil
}

func (s *Store) GetRoom(idStr string) (*models.Room, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	r := &models.Room{}
	query := `SELECT id, created_at FROM rooms WHERE id = $1`
	err = s.db.QueryRow(query, id).Scan(&r.ID, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}
