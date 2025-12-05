FROM golang:1.21-alpine

WORKDIR /app

# Устанавливаем git (нужен для скачивания зависимостей)
RUN apk add --no-cache git

# 1. Сначала копируем ВЕСЬ исходный код
COPY . .

# 2. Теперь, когда код на месте, запускаем tidy.
# Он просканирует папки cmd/ и internal/, найдет импорты и скачает библиотеки.
RUN go mod tidy

# 3. Собираем приложение
RUN go build -o main ./cmd/server

EXPOSE 8080

CMD ["./main"]