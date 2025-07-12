# Knowledge Validation Service API

## 📋 Общая информация

- **Base URL**: `http://localhost:8080/kvs/v1`
- **Content-Type**: `application/json`
- **API Version**: 1.0

## 🔗 Endpoints

### 1. Получение списка тем

**GET** `/topics`

Возвращает список всех доступных тем для тестирования.

#### Ответ
```json
{
  "topics": [
    "Базы данных",
    "Go базовые типы",
    "Алгоритмы и структуры данных"
  ]
}
```

#### Коды ответов
- `200` - Успешно получен список тем
- `404` - Темы не найдены
- `500` - Внутренняя ошибка сервера

---

### 2. Создание новой сессии

**POST** `/{user_id}/start_session`

Создает новую тестовую сессию для пользователя с выбранными темами.

#### Параметры пути
- `user_id` (integer, required) - ID пользователя

#### Тело запроса
```json
{
  "topics": [
    "Базы данных",
    "Go базовые типы"
  ]
}
```

#### Ответ
```json
{
  "session_id": 1234567890,
  "topics": [
    "Базы данных",
    "Go базовые типы"
  ],
  "questions": {
    "1": {
      "id": 1,
      "type": "single selection",
      "topic": "Базы данных",
      "subject": "Что такое ACID?",
      "variants": [
        "Atomicity, Consistency, Isolation, Durability",
        "Availability, Consistency, Isolation, Durability",
        "Atomicity, Concurrency, Isolation, Durability"
      ]
    },
    "2": {
      "id": 2,
      "type": "multi selection",
      "topic": "Go базовые типы",
      "subject": "Какие из перечисленных типов являются базовыми в Go?",
      "variants": [
        "int",
        "string",
        "bool",
        "array"
      ]
    }
  }
}
```

#### Коды ответов
- `201` - Сессия успешно создана
- `400` - Неверные параметры запроса
- `404` - Темы не найдены
- `500` - Внутренняя ошибка сервера

---

### 3. Завершение сессии

**POST** `/{user_id}/{session_id}/complete_session`

Завершает тестовую сессию, отправляя ответы пользователя и получая результаты.

#### Параметры пути
- `user_id` (integer, required) - ID пользователя
- `session_id` (integer, required) - ID сессии

#### Тело запроса
```json
{
  "user_answer": [
    {
      "question_id": 1,
      "answers": ["Atomicity, Consistency, Isolation, Durability"]
    },
    {
      "question_id": 2,
      "answers": ["int", "string", "bool"]
    }
  ]
}
```

#### Ответ
```json
{
  "is_success": true,
  "grade": "75.00 percents"
}
```

#### Коды ответов
- `200` - Сессия успешно завершена
- `400` - Неверные параметры запроса
- `404` - Сессия не найдена
- `500` - Внутренняя ошибка сервера

## 📊 Модели данных

### TopicsDTO
```json
{
  "topics": ["string"]
}
```

### SessionDTO
```json
{
  "session_id": "integer",
  "topics": ["string"],
  "questions": {
    "question_id": {
      "id": "integer",
      "type": "string",
      "topic": "string", 
      "subject": "string",
      "variants": ["string"]
    }
  }
}
```

### UserAnswersListDTO
```json
{
  "user_answer": [
    {
      "question_id": "integer",
      "answers": ["string"]
    }
  ]
}
```

### SessionResultDTO
```json
{
  "is_success": "boolean",
  "grade": "string"
}
```

### ErrorDTO
```json
{
  "status_code": "integer",
  "error_message": "string"
}
```

## 🔄 Жизненный цикл сессии

1. **Создание сессии** - `POST /{user_id}/start_session`
   - Сессия переходит в состояние `InitState`
   - Загружаются вопросы по выбранным темам
   - Сессия переходит в состояние `ActiveState`
   - Запускается таймер

2. **Активная сессия** - состояние `ActiveState`
   - Пользователь отвечает на вопросы
   - Контролируется время выполнения
   - Сессия может истечь по времени

3. **Завершение сессии** - `POST /{user_id}/{session_id}/complete_session`
   - Принимаются ответы пользователя
   - Сессия переходит в состояние `CompletedState`
   - Подсчитываются результаты
   - Возвращается оценка

## ⏱️ Ограничения по времени

- **Время по умолчанию**: 10 минут на тему
- **Проверка истечения**: при отправке ответов
- **Поведение при истечении**: сессия завершается с результатом "session expired"

## 🎯 Система оценок

- **Проходной балл**: 60%
- **Расчет**: (правильные ответы / общее количество вопросов) × 100%
- **Формат результата**: "XX.XX percents"
- **Успех**: `is_success: true` при >= 60%

## 🔍 Типы вопросов

1. **Single Selection** - одиночный выбор
   - Один правильный ответ
   - Пользователь выбирает один вариант

2. **Multi Selection** - множественный выбор
   - Несколько правильных ответов
   - Пользователь может выбрать несколько вариантов

3. **True or False** - да/нет
   - Булевый ответ
   - Пользователь выбирает true или false

## 🚨 Обработка ошибок

Все ошибки возвращаются в стандартном формате:

```json
{
  "status_code": 400,
  "error_message": "Описание ошибки"
}
```

### Типичные ошибки:
- `400` - Неверные параметры запроса
- `404` - Ресурс не найден
- `500` - Внутренняя ошибка сервера

## 📝 Примеры использования

### Полный цикл тестирования

1. **Получение тем**:
```bash
curl -X GET http://localhost:8080/kvs/v1/topics
```

2. **Создание сессии**:
```bash
curl -X POST http://localhost:8080/kvs/v1/123/start_session \
  -H "Content-Type: application/json" \
  -d '{"topics": ["Go базовые типы"]}'
```

3. **Завершение сессии**:
```bash
curl -X POST http://localhost:8080/kvs/v1/123/1234567890/complete_session \
  -H "Content-Type: application/json" \
  -d '{"user_answer": [{"question_id": 1, "answers": ["int"]}]}'
```

## 📚 Дополнительная документация

- [Архитектура системы](ARCHITECTURE.md)
- [Руководство по развертыванию](DEPLOYMENT.md)
- [Swagger спецификация](../api/http/public/swagger.json)
