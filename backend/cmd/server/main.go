package main

import (
	"dice-roller/internal/mw"
	"dice-roller/internal/store"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Конфигурация
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Значение по умолчанию для локального запуска без Docker
		dbURL = "postgres://dice_user:dice_password@localhost:5432/dice_db?sslmode=disable"
	}

	// БД
	storage, err := store.New(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer storage.Close()

	// Роутер
	r := chi.NewRouter()

	// Базовые middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS (разрешаем запросы с фронтенда + передачу куки credentials)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"}, // Порт Vite по умолчанию
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Важно для кук!
	}))

	// API Group
	r.Route("/api", func(r chi.Router) {
		// Применяем AuthMiddleware ко всем роутам API
		r.Use(mw.AuthMiddleware(storage))

		r.Post("/room", handleCreateRoom(storage))
		r.Get("/room/{id}", handleGetRoom(storage))

		// Тестовый роут, чтобы проверить, кто я
		r.Get("/me", handleMe(storage))
	})

	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}

// Handlers

func handleCreateRoom(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		room, err := s.CreateRoom()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Получаем ID пользователя из контекста (если нужно привязать создателя к комнате)
		userID := r.Context().Value(mw.UserIDKey)
		log.Printf("Room created by user: %v", userID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(room)
	}
}

func handleGetRoom(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		room, err := s.GetRoom(id)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(room)
	}
}

func handleMe(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(mw.UserIDKey)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("You are user ID: %v", userID)))
	}
}
