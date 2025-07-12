# Руководство по разработке Knowledge Validation Service

## 🛠️ Настройка среды разработки

### Предварительные требования

- **Go 1.24.3+**
- **PostgreSQL 13+**
- **Git**
- **IDE**: VS Code, GoLand, или любой другой с поддержкой Go

### Установка инструментов

```bash
# Установка Task
go install github.com/go-task/task/v3/cmd/task@latest

# Установка линтера
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Установка Swagger генератора
go install github.com/swaggo/swag/cmd/swag@latest

# Установка gomock для генерации моков
go install github.com/golang/mock/mockgen@latest

# Установка migrate для работы с миграциями
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Клонирование и настройка

```bash
git clone <repository-url>
cd study_platform/question

# Установка зависимостей
go mod download

# Настройка pre-commit хуков
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

## 📁 Структура проекта

```
question/
├── api/                    # API спецификации
│   └── http/public/        # Swagger документация
├── cmd/                    # Точки входа приложения
│   └── question_service/
│       └── main.go
├── deploy/                 # Конфигурации развертывания
│   └── migrations/postgres/
├── docs/                   # Документация
├── internal/               # Внутренняя логика (private)
│   ├── adapter/           # Внешние адаптеры
│   │   ├── generator/     # ID генераторы
│   │   └── storage/       # Адаптеры хранилища
│   ├── cases/             # Use Cases (бизнес-логика)
│   ├── entities/          # Доменные сущности
│   └── port/              # Порты (интерфейсы)
│       └── http/public/   # HTTP API
└── pkg/                   # Публичные пакеты
    └── dto/               # Data Transfer Objects
```

## 🏗️ Архитектурные принципы

### Clean Architecture

Проект следует принципам Clean Architecture:

1. **Entities** (`internal/entities/`) - бизнес-логика
2. **Use Cases** (`internal/cases/`) - прикладная логика
3. **Interface Adapters** (`internal/adapter/`, `internal/port/`) - адаптеры
4. **Frameworks & Drivers** - внешние библиотеки

### Dependency Rule

Зависимости направлены внутрь:
- Entities не зависят ни от чего
- Use Cases зависят только от Entities
- Adapters зависят от Use Cases и Entities
- Frameworks зависят от всех слоев

## 🔧 Команды разработки

### Основные команды Task

```bash
# Запуск линтера
task lint

# Запуск PostgreSQL
task postgres:up
task postgres:stop
task postgres:restart

# Миграции
task postgres:migrate:up
task postgres:migrate:down

# Тестирование
task l1_test

# Генерация Swagger
task swag_gen
```

### Команды Go

```bash
# Запуск приложения
go run ./cmd/question_service/main.go

# Сборка
go build -o bin/kvs ./cmd/question_service

# Тестирование
go test ./...
go test -v ./internal/...

# Генерация моков
go generate ./...
```

## 🧪 Тестирование

### Структура тестов

```
internal/
├── entities/
│   ├── session.go
│   ├── session_test.go
│   └── testdata/
│       └── mocks.go
├── cases/
│   ├── session_service.go
│   ├── session_service_test.go
│   └── testdata/
└── adapter/
    └── storage/postgres/
        ├── storage.go
        └── storage_test.go
```

### Типы тестов

1. **Unit тесты** - тестирование отдельных функций
2. **Integration тесты** - тестирование с базой данных
3. **L1 тесты** - комплексное тестирование

### Написание тестов

```go
func TestSessionService_CreateSession(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockStorage := testdata.NewMockStorage(ctrl)
    service := NewSessionService(mockStorage, /* ... */)
    
    // Act
    result, err := service.CreateSession(context.Background(), 1, []string{"Go"})
    
    // Assert
    require.NoError(t, err)
    assert.NotZero(t, result)
}
```

### Моки

Генерация моков происходит автоматически через `go generate`:

```go
//go:generate mockgen -source=storage.go -destination=./testdata/storage.go -package=testdata
type Storage interface {
    GetTopics(ctx context.Context) ([]string, error)
    // ...
}
```

## 📝 Стандарты кодирования

### Именование

- **Пакеты**: короткие, описательные имена (`entities`, `cases`)
- **Интерфейсы**: существительные (`Storage`, `Question`)
- **Функции**: глаголы (`CreateSession`, `GetTopics`)
- **Переменные**: camelCase (`userID`, `sessionResult`)

### Структура файлов

```go
package entities

import (
    // Стандартные библиотеки
    "context"
    "time"
    
    // Внешние зависимости
    "github.com/pkg/errors"
    
    // Внутренние пакеты
    "github.com/parta4ok/kvs/question/pkg/dto"
)

// Константы
const (
    DefaultTimeout = time.Minute * 5
)

// Типы
type Session struct {
    // ...
}

// Конструкторы
func NewSession(...) (*Session, error) {
    // ...
}

// Методы
func (s *Session) GetID() uint64 {
    // ...
}
```

### Обработка ошибок

```go
// Доменные ошибки
var (
    ErrInvalidParam = errors.New("invalid parameter")
    ErrNotFound     = errors.New("not found")
)

// Обертывание ошибок
func (s *Service) CreateSession(ctx context.Context, userID uint64) error {
    session, err := s.storage.GetSession(ctx, userID)
    if err != nil {
        return errors.Wrapf(err, "failed to get session for user %d", userID)
    }
    // ...
}
```

## 🔄 Workflow разработки

### Git Flow

1. **Feature branches**: `feature/add-new-question-type`
2. **Bug fixes**: `fix/session-timeout-issue`
3. **Releases**: `release/v1.2.0`

### Процесс разработки

1. **Создание ветки**:
```bash
git checkout -b feature/new-feature
```

2. **Разработка**:
```bash
# Написание кода
# Написание тестов
task lint
task l1_test
```

3. **Коммит**:
```bash
git add .
git commit -m "feat: add new question type support"
```

4. **Push и PR**:
```bash
git push origin feature/new-feature
# Создание Pull Request
```

### Commit Messages

Используем Conventional Commits:

- `feat:` - новая функциональность
- `fix:` - исправление бага
- `docs:` - изменения в документации
- `test:` - добавление тестов
- `refactor:` - рефакторинг кода

## 🚀 Добавление новых функций

### Добавление нового типа вопроса

1. **Создание сущности**:
```go
// internal/entities/new_question_type.go
type NewQuestionType struct {
    // поля
}

func (q *NewQuestionType) IsAnswerCorrect(ans *UserAnswer) bool {
    // логика проверки
}
```

2. **Обновление фабрики**:
```go
// internal/entities/question.go
func (factory *QuestionFactory) NewQuestion(...) (Question, error) {
    switch questionType {
    case NewType:
        return NewNewQuestionType(...), nil
    }
}
```

3. **Тесты**:
```go
// internal/entities/new_question_type_test.go
func TestNewQuestionType_IsAnswerCorrect(t *testing.T) {
    // тесты
}
```

### Добавление нового эндпоинта

1. **Обновление интерфейса сервиса**:
```go
// internal/port/http/public/service.go
type Service interface {
    NewMethod(ctx context.Context, param string) (Result, error)
}
```

2. **Реализация в сервисе**:
```go
// internal/cases/session_service.go
func (s *SessionService) NewMethod(ctx context.Context, param string) (Result, error) {
    // реализация
}
```

3. **HTTP handler**:
```go
// internal/port/http/public/server.go
func (s *Server) NewEndpoint(resp http.ResponseWriter, req *http.Request) {
    // обработка HTTP запроса
}
```

4. **Регистрация маршрута**:
```go
func (s *Server) registerRoutes() {
    s.router.Post("/new-endpoint", s.NewEndpoint)
}
```

## 🔍 Отладка

### Логирование

```go
import "log/slog"

// Структурированное логирование
slog.Info("Session created", 
    "user_id", userID, 
    "session_id", sessionID,
    "topics", topics)

slog.Error("Failed to create session", 
    "error", err,
    "user_id", userID)
```

### Профилирование

```go
import _ "net/http/pprof"

// Добавление pprof эндпоинтов
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

### Отладка базы данных

```bash
# Подключение к БД
psql -h localhost -p 5432 -U postgres knowledge

# Просмотр активных сессий
SELECT * FROM kvs.sessions WHERE state = 'active state';

# Анализ производительности
EXPLAIN ANALYZE SELECT * FROM kvs.questions WHERE topic_id = 1;
```

## 📚 Полезные ресурсы

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)

## 🤝 Contribution Guidelines

1. Форкните репозиторий
2. Создайте feature ветку
3. Следуйте стандартам кодирования
4. Добавьте тесты для новой функциональности
5. Убедитесь, что все тесты проходят
6. Создайте Pull Request с описанием изменений
