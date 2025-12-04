package models

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

type CreateRoomRequest struct {
	// Можно добавить настройки комнаты (макс игроков и т.д.)
}
