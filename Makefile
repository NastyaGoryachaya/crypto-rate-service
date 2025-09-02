APP_NAME=crypto-rate-service
CMD_DIR=cmd/app
BINARY=bin/$(APP_NAME)
DOCKER_IMAGE=$(APP_NAME):latest
MIGRATIONS_PATH=./migrations
DB_URL=postgres://postgres:postgres@localhost:5432/crypto?sslmode=disable

.PHONY: run build tidy fmt lint test docker-build docker-up docker-down migrate-up migrate-down migrate-create docker-logs

# Запуск сервиса локально
run:
	go run $(CMD_DIR)/main.go

# Сборка бинарника
build:
	go build -o $(BINARY) $(CMD_DIR)/main.go

# Очистка зависимостей
tidy:
	go mod tidy

# Форматирование кода
fmt:
	go fmt ./...

# Линтер (требуется golangci-lint)
lint:
	golangci-lint run

# Запуск тестов
test:
	go test ./... -v

# Миграции
migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1

migrate-create:
	@read -p "Введите имя миграции: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_PATH) $$name

# Сборка Docker-образа
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Запуск контейнеров (docker-compose.yml)
docker-up:
	docker-compose up -d

# Остановка контейнеров
docker-down:
	docker-compose down

# Просмотр логов контейнеров
 docker-logs:
	docker-compose logs -f