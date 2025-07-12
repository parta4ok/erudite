# Руководство по развертыванию Knowledge Validation Service

## 🚀 Обзор развертывания

Knowledge Validation Service может быть развернут в различных окружениях: локально для разработки, в контейнерах Docker, или в облачных платформах.

## 📋 Предварительные требования

### Системные требования
- **Go**: версия 1.24.3 или выше
- **PostgreSQL**: версия 13 или выше
- **Task**: для выполнения команд сборки
- **Docker**: для контейнеризации (опционально)
- **Docker Compose**: для оркестрации (опционально)

### Инструменты разработки
```bash
# Установка Task
go install github.com/go-task/task/v3/cmd/task@latest

# Установка golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Установка swag для генерации Swagger
go install github.com/swaggo/swag/cmd/swag@latest

# Установка golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## 🏠 Локальное развертывание

### 1. Подготовка окружения

```bash
# Клонирование репозитория
git clone <repository-url>
cd study_platform/question

# Создание .env файла
cp .env.example .env
```

### 2. Настройка базы данных

```bash
# Запуск PostgreSQL через Docker Compose
task postgres:up

# Применение миграций
task postgres:migrate:up

# Проверка подключения
psql -h localhost -p 5432 -U postgres -d knowledge
```

### 3. Сборка и запуск

```bash
# Сборка приложения
go build -o bin/kvs ./cmd/question_service

# Запуск сервиса (различные способы)
./bin/kvs                                    # Использует конфигурацию по умолчанию
./bin/kvs -config /path/to/config.yaml      # Указание пути к конфигурации
./bin/kvs -help                             # Показать справку

# Через переменную окружения
KVS_CONFIG_PATH=/path/to/config.yaml ./bin/kvs

# Или через go run
go run ./cmd/question_service/main.go
go run ./cmd/question_service/main.go -config ./deployment/question.yaml
```

### 4. Проверка работоспособности

```bash
# Проверка API
curl http://localhost:8080/kvs/v1/topics

# Генерация Swagger документации
task swag_gen

# Запуск тестов
task l1_test
```

## 🐳 Docker развертывание

### Dockerfile

```dockerfile
# Многоэтапная сборка
FROM golang:1.24.3-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o kvs ./cmd/question_service

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/kvs .
COPY --from=builder /app/deploy/migrations ./migrations

EXPOSE 8080
CMD ["./kvs"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  kvs:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=password
      - DB_NAME=knowledge
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=knowledge
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./deploy/migrations/postgres:/docker-entrypoint-initdb.d
    restart: unless-stopped

volumes:
  postgres_data:
```

### Команды Docker

```bash
# Сборка образа
docker build -t kvs:latest .

# Запуск через Docker Compose
docker-compose up -d

# Просмотр логов
docker-compose logs -f kvs

# Остановка
docker-compose down
```

## ☁️ Облачное развертывание

### Kubernetes

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kvs-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kvs
  template:
    metadata:
      labels:
        app: kvs
    spec:
      containers:
      - name: kvs
        image: kvs:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_PORT
          value: "5432"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"

---
apiVersion: v1
kind: Service
metadata:
  name: kvs-service
spec:
  selector:
    app: kvs
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 🔧 Конфигурация

### Конфигурационный файл

Приложение использует YAML файл для конфигурации. По умолчанию ищется файл `deployment/question.yaml`.

#### Структура конфигурации

```yaml
kvs:
  http:
    public:
      port: :8080
      timeout: 30s
  storage:
    type: postgres
postgres:
  connection: postgresql://postgres:password@localhost:5432/knowledge
```

#### Способы указания конфигурации

1. **По умолчанию**: `deployment/question.yaml`
2. **Флаг командной строки**: `-config /path/to/config.yaml`
3. **Переменная окружения**: `KVS_CONFIG_PATH=/path/to/config.yaml`

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `KVS_CONFIG_PATH` | Путь к конфигурационному файлу | `deployment/question.yaml` |
| `DB_HOST` | Хост базы данных | `localhost` |
| `DB_PORT` | Порт базы данных | `5432` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль БД | `password` |
| `DB_NAME` | Имя базы данных | `knowledge` |
| `SERVER_PORT` | Порт HTTP сервера | `8080` |
| `LOG_LEVEL` | Уровень логирования | `info` |

### Пример .env файла

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=knowledge

# Server
SERVER_PORT=8080
LOG_LEVEL=info

# Testing
TESTPGCONN=postgresql://postgres:password@localhost:5432/knowledge?sslmode=disable
```

## 🗄️ Управление базой данных

### Миграции

```bash
# Применение всех миграций
task postgres:migrate:up

# Откат всех миграций
task postgres:migrate:down

# Создание новой миграции
migrate create -ext sql -dir deploy/migrations/postgres -seq new_migration_name
```

### Резервное копирование

```bash
# Создание бэкапа
pg_dump -h localhost -p 5432 -U postgres knowledge > backup.sql

# Восстановление из бэкапа
psql -h localhost -p 5432 -U postgres knowledge < backup.sql
```

## 📊 Мониторинг и логирование

### Логирование

Сервис использует структурированное логирование через `slog`:

```go
// Пример настройки логирования
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)
```

### Метрики

Для мониторинга рекомендуется использовать:
- **Prometheus** - сбор метрик
- **Grafana** - визуализация
- **Jaeger** - трассировка запросов

### Health Check

```bash
# Проверка состояния сервиса
curl http://localhost:8080/kvs/v1/topics

# Проверка базы данных
psql -h localhost -p 5432 -U postgres -c "SELECT 1" knowledge
```

## 🔒 Безопасность

### Рекомендации по безопасности

1. **Переменные окружения**: Не храните пароли в коде
2. **HTTPS**: Используйте TLS в production
3. **Firewall**: Ограничьте доступ к базе данных
4. **Обновления**: Регулярно обновляйте зависимости

### Пример nginx конфигурации

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 🧪 Тестирование в production

```bash
# Smoke тесты
curl -f http://your-domain.com/kvs/v1/topics || exit 1

# Load тестирование
ab -n 1000 -c 10 http://your-domain.com/kvs/v1/topics
```

## 🚨 Устранение неполадок

### Частые проблемы

1. **Подключение к БД**: Проверьте переменные окружения
2. **Порт занят**: Измените SERVER_PORT
3. **Миграции**: Убедитесь, что миграции применены
4. **Права доступа**: Проверьте права пользователя БД

### Логи для отладки

```bash
# Docker логи
docker-compose logs -f kvs

# Системные логи
journalctl -u kvs -f

# Логи базы данных
docker-compose logs -f postgres
```
