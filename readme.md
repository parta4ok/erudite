# Erudite — Study & Self-Testing Platform

**Erudite** — это open-source платформа для самопроверки и персонального обучения студентов, поддерживающая сценарии генерации тестов, отслеживания прогресса и анализа пробелов в знаниях.

## 🚀 Quick Start

### Backend Services

```bash
# Start all backend services
task services:run
```

### Frontend Application

The frontend has its own separate setup and Docker configuration:

```bash
cd frontend/web-app
make start
# Or manually:
npm install
npm run dev
```

For Docker deployment:

```bash
cd frontend/web-app
docker-compose up --build
```

Access the application at:

- **Frontend**: http://localhost:3000
- **Auth API**: http://localhost:8090
- **Question API**: http://localhost:8080
- **Reporting API**: http://localhost:8082

### Default Login

- **Email**: `admin@kvs.ru`
- **Password**: `password123`

---

## Задачи

- Внедрение в процесс обучения с целью минимизации временных затрат на проверку базовых знаний студента

## 🏗️ Architecture

### Backend Services (Go)

- **Auth Service** - Authentication and user management
- **Question Service** - Session and topic management
- **Reporting Service** - Analytics and progress tracking
- **NotificationHub** - Email and Telegram notifications

### Frontend (React + TypeScript)

- **Separate Docker Setup** - Independent docker-compose.yml in `frontend/web-app/`
- **Modern Stack** - React 18, TypeScript, Vite, React Query
- **Admin Panel** - User and group management
- **Session Management** - Topic selection and question answering (planned)

> **Note**: Frontend runs independently from backend services and has its own Docker configuration to avoid conflicts.

- **Web Application** - Modern React-based UI
- **Admin Panel** - User and group management
- **Student Dashboard** - Learning sessions and progress

## Обобзенный сценарий тестирования для студента

```mermaid
sequenceDiagram
    actor S as Student
    participant A as Auth Service
    participant Q as Question Service
    participant N as NotificationHub Service
    actor M as Mentor

    S ->> A: login
    activate A
    A -->> S: jwt
    deactivate A

    S ->> Q: GetTopics(jwt)
    activate Q
    Q ->> A: introspect(jwt)
    activate A
    A -->> Q: user claims
    deactivate A
    ALT user has not enough rights
        Q -->> S: forbidden
    END
    Q -->>S: existings topics
    deactivate Q

    S ->>S: select topic/topics from existings topics

    S ->> Q: StartSession(jwt, topics)
    activate Q
    Q ->> A: introspect(jwt)
    activate A
    A -->> Q: user claims
    deactivate A
    ALT user has not enough rights
        Q -->> S: forbidden
    END
    Q -->> S: QuestionsSession (questions from selected topics)
    deactivate Q
    S ->> S: select and set questions answers

    S ->> Q: answers(jwt)
    activate Q
    Q ->> Q: generate session result
    Q ->> N: session result
    activate N
    Q -->> S: session result
    deactivate Q

    N ->> A: get recipient (studentID)
    activate A
    A -->> N: recipient id, recipient contacts
    deactivate A
    Loop notifiers
        N ->> M: try to send session result by concrete notifier like telegram, email, etc

    END
    deactivate N
```

## Возможности

- Автоматическая генерация тестовых сессий по выбранным темам
- Поддержка разных типов вопросов: одиночный выбор; множественный выбор; true/false
- Ограничение времени на выполнение сессии
- Расчет оценки и настройка порога успешной сдачи сессии
- Создание/удаление пользователей, наделение пользователей правами

## Контракты

- Open API/gRPC спецификации сервисов можно найти в папке API в корне проекта: './api'

## Особенности

- Используйте миграции для создания новых топиков и вопросов, аналогично, для создания первоначального администратора и/или других пользователей
- Реализована простая система аутентификации и авторизации
- Внедрены роли (набор прав на выполнение определенных операций)
- Существует всего 3 роли: Администратор, Ментор, Студент
- Только Администратор имеет право на создание и удаление новых пользователей
- Администратор и Ментор имеют права на получение данных по всем завершенным сессиям
- Все типы пользователей имеют права на просмотр топико, запуск сессий и их завершение
- Код покрыт юнит-тестами, L1-тестами (используется контейнер с БД), L2-тестами (все контейнеры задействуются)

## Стек и технологии

- Основной язык - Go ver 1.24
- База данных - PostgreSQL ver 16
- Линтер - golangci-lint ver 2
- Контейнеризация - Docker
- Межсетевое взаимодействие - REST API, gRPC
- CI - Github Actions
- Автоматизация - Task

## Планы

- Внедрение сервиса оповещения для менторов с использованием телеграм
- Расширение observability, в т.ч добавление распределенных трассировок, мониторинга метрик
- Расширение контрактов для возможности быстрого добавления вопросов и тем
- Адаптация базового сценария использования сервиса в телеграм

# Контакты

## e-mail: parta4ok@google.com
