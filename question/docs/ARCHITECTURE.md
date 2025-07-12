# Архитектура Knowledge Validation Service

## 🏛️ Общий обзор архитектуры

Knowledge Validation Service построен на принципах **Clean Architecture** (Чистая архитектура) с четким разделением ответственности между слоями.

## 📐 Слои архитектуры

### 1. Entities (Доменные сущности)
**Расположение**: `internal/entities/`

Содержит основную бизнес-логику и доменные модели:

- **Session** - управление тестовыми сессиями
- **Question** - интерфейс для различных типов вопросов
- **UserAnswer** - пользовательские ответы
- **SessionState** - состояния сессий (State Machine)

#### Основные сущности:

```go
// Session - основная сущность для управления тестированием
type Session struct {
    userID    uint64
    sessionID uint64
    topics    []string
    state     SessionState
}

// Question - интерфейс для всех типов вопросов
type Question interface {
    ID() uint64
    Type() QuestionType
    Topic() string
    Subject() string
    Variants() []string
    IsAnswerCorrect(ans *UserAnswer) bool
}
```

### 2. Use Cases (Прикладная логика)
**Расположение**: `internal/cases/`

Содержит бизнес-логику приложения:

- **SessionService** - управление сессиями тестирования
- **Storage** - интерфейс для работы с хранилищем

#### Основные сервисы:

```go
type SessionService struct {
    storage        Storage
    sessionStorage entities.SessionStorage
    generator      entities.IDGenerator
    topicDuration  time.Duration
}
```

### 3. Interface Adapters (Адаптеры интерфейсов)
**Расположение**: `internal/adapter/` и `internal/port/`

#### Адаптеры (`internal/adapter/`):
- **PostgreSQL Storage** - адаптер для работы с PostgreSQL
- **ID Generator** - генератор уникальных идентификаторов

#### Порты (`internal/port/`):
- **HTTP Server** - REST API сервер
- **Service Interface** - интерфейс для бизнес-логики

### 4. Frameworks & Drivers (Фреймворки и драйверы)
**Расположение**: `cmd/`, внешние библиотеки

- HTTP сервер (Chi router)
- PostgreSQL драйвер (pgx/v5)
- Логирование (slog)

## 🔄 State Machine для сессий

Сессии управляются через паттерн State Machine с тремя состояниями:

1. **InitState** - начальное состояние
2. **ActiveState** - активная сессия с вопросами
3. **CompletedState** - завершенная сессия с результатами

```
InitState → ActiveState → CompletedState
     ↓           ↓             ↓
  SetQuestions SetUserAnswer GetResult
```

## 🎯 Типы вопросов

Система поддерживает различные типы вопросов через полиморфизм:

1. **SingleSelectionQuestion** - одиночный выбор
2. **MultiSelectionQuestion** - множественный выбор  
3. **TrueOrFalseQuestion** - да/нет

Все типы реализуют интерфейс `Question`.

## 🗄️ Модель данных

### Основные таблицы:

- **kvs.topics** - темы для тестирования
- **kvs.questions** - вопросы с метаданными
- **kvs.question_types** - типы вопросов
- **kvs.sessions** - пользовательские сессии

### Связи:
- Question → Topic (many-to-one)
- Question → QuestionType (many-to-one)
- Session → Questions (many-to-many через JSON)

## 🔌 Dependency Injection

Проект использует функциональные опции для внедрения зависимостей:

```go
// Пример создания сервера
server, err := public.New(
    public.WithService(sessionService),
    public.WithConfig(&public.ServerCfg{Port: ":8080"}),
)
```

## 🚦 Поток данных

### Создание сессии:
1. HTTP запрос → Server
2. Server → SessionService
3. SessionService → Storage (получение вопросов)
4. SessionService → Session (создание с InitState)
5. Session → ActiveState (установка вопросов)
6. Storage ← Session (сохранение)
7. HTTP ответ ← Server

### Завершение сессии:
1. HTTP запрос → Server
2. Server → SessionService
3. SessionService → Storage (получение сессии)
4. Session → CompletedState (установка ответов)
5. Session → результат (подсчет оценки)
6. Storage ← Session (сохранение)
7. HTTP ответ ← Server

## 🛡️ Обработка ошибок

Система использует централизованную обработку ошибок:

- **Доменные ошибки** - определены в entities
- **Обертывание ошибок** - через pkg/errors
- **HTTP коды** - автоматическое сопоставление
- **Логирование** - структурированные логи

## 🔧 Конфигурация

Конфигурация осуществляется через:
- Переменные окружения
- Функциональные опции
- Значения по умолчанию

## 📊 Мониторинг

- Структурированное логирование (slog)
- Отслеживание времени выполнения
- Мониторинг состояний сессий
- Логирование ошибок базы данных

## 🧪 Тестируемость

Архитектура обеспечивает высокую тестируемость:

- **Интерфейсы** - для всех внешних зависимостей
- **Моки** - автогенерация через gomock
- **Изоляция** - каждый слой тестируется независимо
- **Integration тесты** - с реальной базой данных

## 🚀 Масштабируемость

Архитектура поддерживает горизонтальное масштабирование:

- **Stateless сервис** - состояние в базе данных
- **Пулинг соединений** - эффективное использование БД
- **Graceful shutdown** - корректное завершение
- **Микросервисная готовность** - четкие границы

## 🔄 Расширяемость

Легко добавлять новые функции:

- **Новые типы вопросов** - через интерфейс Question
- **Новые адаптеры** - через интерфейсы Storage
- **Новые состояния** - через SessionState
- **Новые эндпоинты** - через HTTP handlers
