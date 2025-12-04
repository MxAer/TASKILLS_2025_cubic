package mw

import (
	"context"
	"dice-roller/internal/store"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type key int

const (
	UserIDKey key = iota
)

func AuthMiddleware(storage *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("dice_session")
			var userID uuid.UUID

			// Если куки нет или она невалидна
			if err != nil || cookie.Value == "" {
				// 1. Создаем нового пользователя в БД
				user, err := storage.CreateUser("Guest")
				if err != nil {
					http.Error(w, "Failed to register guest", http.StatusInternalServerError)
					return
				}
				userID = user.ID

				// 2. Ставим куку
				http.SetCookie(w, &http.Cookie{
					Name:     "dice_session",
					Value:    userID.String(),
					Path:     "/",
					Expires:  time.Now().Add(24 * 30 * time.Hour), // 30 дней
					HttpOnly: true,                                // Недоступно из JS (безопасность)
					SameSite: http.SameSiteLaxMode,
				})
			} else {
				// Парсим ID из куки
				parsedID, err := uuid.Parse(cookie.Value)
				if err != nil {
					// Если кука битая - сбрасываем (в реальном проекте можно пересоздать)
					http.Error(w, "Invalid session", http.StatusBadRequest)
					return
				}
				userID = parsedID
			}

			// Кладем ID пользователя в контекст запроса
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
