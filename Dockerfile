# Используем официальный образ Go для сборки
FROM golang:1.21 AS builder

WORKDIR /app

# Копируем код
COPY . .

# Собираем бинарник
RUN go mod tidy && go build -o bot .

# Используем минимальный образ для работы
FROM debian:bullseye-slim

WORKDIR /app

# Копируем скомпилированный бинарник
COPY --from=builder /app/bot .

# Запускаем бота
CMD ["./bot"]
