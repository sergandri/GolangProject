# Используйте официальный образ Go как родительский образ
FROM golang:1.22-alpine

LABEL authors="serga"

# Установка рабочего каталога в контейнере
WORKDIR /app

# Копирование файла go.mod и, если есть, go.sum, и загрузка зависимостей
COPY go.* ./
RUN go mod download

# Копирование исходного кода проекта в контейнер
COPY . .

# Сборка приложения
RUN go build -o /transactions_server

# Определение порта, который будет прослушивать приложение
EXPOSE 8080

# Запуск скомпилированного бинарного файла
CMD ["/transactions_server"]
