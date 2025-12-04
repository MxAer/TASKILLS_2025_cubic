CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Таблица пользователей (гостей)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) DEFAULT 'Guest',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица комнат
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Таблица бросков (понадобится позже, но создаем сейчас)
CREATE TABLE IF NOT EXISTS rolls (
    id SERIAL PRIMARY KEY,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    dice_config VARCHAR(50) NOT NULL,
    result_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
