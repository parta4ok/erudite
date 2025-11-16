# Архитектура NotificationHub Service

## Введение
NotificationHub Service (далее - сервис уведомлений) является специализированным микросервисом в составе проекта Knowledge Validation System (далее - kvs, erudite), отвечающим за доставку уведомлений о результатах тестирования пользователям системы.

Сервис позволяет:
- Принимать результаты сессий тестирования через NATS брокер;
- Получать информацию о связанных пользователях из Auth Service;
- Отправлять уведомления через различные каналы связи (Telegram, Email);
- Обеспечивать надежную доставку сообщений с использованием паттерна Chain of Responsibility.

## System Design
Сервис построен на основе событийно-ориентированной архитектуры и использует асинхронную обработку сообщений. 
Сервис написан на языке Go.
В качестве точки входа используется NATS Consumer для получения результатов сессий.
Для получения информации о пользователях используется gRPC клиент для обращения к Auth Service.
Для отправки уведомлений реализована цепочка нотификаторов с использованием паттерна Chain of Responsibility.

### Структура сервиса
```mermaid
graph TD
    subgraph QuestionService[Question Service]
        A[Session Complete Event]
    end

    subgraph NATSBroker[NATS Broker]
        B[(Stream session result)]
    end

    subgraph NotificationHub[NotificationHub Service]
        C[NATS Consumer]
        D[Message Service]
        E[Auth Client]
        F[Notifier Chain]
        G[Telegram Notifier]
        H[Mail Notifier]
    end

    subgraph AuthService[Auth Service]
        I[Get Linked Users]
    end

    subgraph External[External Services]
        J[Telegram Bot API]
        K[SMTP Server]
    end

    A --> |publish| B
    B --> |consume| C
    C --> D
    D --> E
    E --> |gRPC| I
    D --> F
    F --> G
    F --> H
    G --> |send message| J
    H --> |send email| K
```

## Структура проекта

### Диаграмма классов
```mermaid
classDiagram
    %% Core Entities
    class SessionResult {
        -userID string
        -topics strings
        -questions map
        -userAnswer map
        -isExpire bool
        -isSuccess bool
        -resume string
        +NewSessionResult(...) SessionResult
        +GetUserID() string
        +Validate() error
    }
    
    class LinkedUsers {
        -recipient User
        -student User
        +NewLinkedUsers(recipient, student) LinkedUsers
        +GetRecipient() User
        +GetStudent() User
    }
    
    class User {
        -id string
        -name string
        -fullname string
        -rights strings
        -contacts map
        -groupID string
        +GetID() string
        +GetContacts() map
    }
    
    %% Chain of Responsibility Pattern
    class Notifier {
        <<interface>>
        +Notify(sessionResult, linkedUsers) error
        +Next() Notifier
        +SetNextNotifier(notifier)
    }
    
    class TelegramNotifier {
        -bot BotAPI
        -next Notifier
        +NewTelegramNotifier(next, token) TelegramNotifier
        +Notify(sessionResult, linkedUsers) error
        +Next() Notifier
        +SetNextNotifier(notifier)
        -checkTelegramInContacts(linkedUsers) string
        -generateMessage(sessionResult, linkedUsers) string
    }
    
    class MailNotifier {
        -next Notifier
        -host string
        -baseMail string
        -basePort string
        -pwd string
        +NewMailNotifier(next, host, baseMail, basePort, pwd) MailNotifier
        +Notify(sessionResult, linkedUsers) error
        +Next() Notifier
        +SetNextNotifier(notifier)
        -checkMailInContacts(linkedUsers) string
        -generateMessage(sessionResult, linkedUsers) string
    }
    
    %% Use Cases
    class MessageService {
        -notifier Notifier
        -authClient AuthClient
        +NewMessageService(notifier, authClient) MessageService
        +SendMessage(ctx, sessionResult) error
    }
    
    class AuthClient {
        <<interface>>
        +GetLinkedUsers(ctx, id) LinkedUsers
    }
    
    class AuthService {
        -client AuthClient
        +NewAuthService(port) AuthService
        +GetLinkedUsers(ctx, id) LinkedUsers
    }
    
    %% Ports
    class MessageServicePort {
        <<interface>>
        +SendMessage(ctx, sessionResult) error
    }
    
    class NATSConsumer {
        -js JetStreamContext
        -messageService MessageServicePort
        -subscription Subscription
        -subject string
        -ctx Context
        -cancel CancelFunc
        -wg WaitGroup
        +NewNatsConsumer(conn, subject, messageService) NATSConsumer
        +Start() error
        +Stop() error
        -handleMessage(msg) error
    }
    
    %% DTOs
    class SessionResultDTO {
        +UserID string
        +Topics strings
        +Questions map
        +UserAnswer map
        +IsExpire bool
        +IsSuccess bool
        +Resume string
    }
    
    %% Relations - Chain of Responsibility
    TelegramNotifier ..|> Notifier : implements
    MailNotifier ..|> Notifier : implements
    TelegramNotifier --> Notifier : next
    MailNotifier --> Notifier : next
    
    %% Relations - Use Cases
    MessageService --> Notifier : uses
    MessageService --> AuthClient : uses
    AuthService ..|> AuthClient : implements
    MessageService ..|> MessageServicePort : implements
    
    %% Relations - Ports
    NATSConsumer --> MessageServicePort : uses
    NATSConsumer --> SessionResultDTO : receives
    
    %% Relations - Data Flow
    MessageService --> SessionResult : processes
    MessageService --> LinkedUsers : retrieves
    Notifier --> SessionResult : notifies about
    Notifier --> LinkedUsers : sends to
    
    %% Relations - Entities
    LinkedUsers --> User : contains
    SessionResult --> User : belongs to
```

### Диаграммы последовательностей

#### 1. Основной процесс обработки уведомления

```mermaid
sequenceDiagram
    participant QuestionService
    participant NATS
    participant NATSConsumer
    participant MessageService
    participant AuthClient
    participant AuthService
    participant NotifierChain
    participant TelegramNotifier
    participant MailNotifier
    participant TelegramAPI
    participant SMTPServer
    
    QuestionService->>NATS: Publish SessionResult
    NATS->>NATSConsumer: Consume message
    NATSConsumer->>NATSConsumer: Parse SessionResultDTO
    NATSConsumer->>MessageService: SendMessage(ctx, sessionResult)
    
    MessageService->>AuthClient: GetLinkedUsers(ctx, userID)
    AuthClient->>AuthService: gRPC call
    AuthService-->>AuthClient: LinkedUsers
    AuthClient-->>MessageService: LinkedUsers
    
    MessageService->>NotifierChain: Notify(sessionResult, linkedUsers)
    NotifierChain->>TelegramNotifier: Notify(sessionResult, linkedUsers)
    
    alt Telegram contact found
        TelegramNotifier->>TelegramAPI: Send message
        TelegramAPI-->>TelegramNotifier: Success
        TelegramNotifier-->>NotifierChain: Success
    else Telegram contact not found
        TelegramNotifier->>TelegramNotifier: Check next notifier
        TelegramNotifier->>MailNotifier: Notify(sessionResult, linkedUsers)
        
        alt Email contact found
            MailNotifier->>SMTPServer: Send email
            SMTPServer-->>MailNotifier: Success
            MailNotifier-->>TelegramNotifier: Success
        else Email contact not found
            MailNotifier-->>TelegramNotifier: No contacts available
        end
    end
    
    NotifierChain-->>MessageService: Notification result
    MessageService-->>NATSConsumer: Processing complete
```

#### 2. Паттерн Chain of Responsibility в действии

```mermaid
sequenceDiagram
    participant MessageService
    participant TelegramNotifier
    participant MailNotifier
    participant NullNotifier
    
    Note over MessageService,NullNotifier: Chain: Telegram -> Mail -> Null
    
    MessageService->>TelegramNotifier: Notify(sessionResult, linkedUsers)
    TelegramNotifier->>TelegramNotifier: checkTelegramInContacts()
    
    alt Telegram contact exists and valid
        TelegramNotifier->>TelegramNotifier: sendTelegramMessage()
        alt Message sent successfully
            TelegramNotifier-->>MessageService: Success
        else Send failed
            TelegramNotifier->>TelegramNotifier: Next()
            TelegramNotifier->>MailNotifier: Notify(sessionResult, linkedUsers)
            Note over MailNotifier: Fallback to next notifier
        end
    else No Telegram contact
        TelegramNotifier->>TelegramNotifier: Next()
        TelegramNotifier->>MailNotifier: Notify(sessionResult, linkedUsers)
        
        MailNotifier->>MailNotifier: checkMailInContacts()
        alt Email contact exists and valid
            MailNotifier->>MailNotifier: sendEmail()
            alt Email sent successfully
                MailNotifier-->>TelegramNotifier: Success
            else Send failed
                MailNotifier->>MailNotifier: Next()
                MailNotifier->>NullNotifier: Notify(sessionResult, linkedUsers)
                NullNotifier-->>MailNotifier: Log: No more notifiers
            end
        else No Email contact
            MailNotifier->>MailNotifier: Next()
            MailNotifier->>NullNotifier: Notify(sessionResult, linkedUsers)
            NullNotifier-->>MailNotifier: Log: No more notifiers
        end
    end
```

#### 3. Инициализация цепочки нотификаторов

```mermaid
sequenceDiagram
    participant Main
    participant TelegramNotifier
    participant MailNotifier
    participant MessageService
    
    Main->>MailNotifier: NewMailNotifier(nil, config...)
    MailNotifier-->>Main: mailNotifier
    
    Main->>TelegramNotifier: NewTelegramNotifier(mailNotifier, token)
    TelegramNotifier->>TelegramNotifier: SetNextNotifier(mailNotifier)
    TelegramNotifier-->>Main: telegramNotifier
    
    Main->>MessageService: NewMessageService(telegramNotifier, authClient)
    MessageService-->>Main: messageService
    
    Note over Main: Chain configured: Telegram -> Mail -> nil
```

## Паттерны проектирования

### Chain of Responsibility (Цепочка обязанностей)

Сервис уведомлений активно использует паттерн Chain of Responsibility для обеспечения надежной доставки сообщений через различные каналы связи.

#### Структура паттерна:

1. **Интерфейс Handler (Notifier)**:
   ```go
   type Notifier interface {
       Notify(sessionResult *entities.SessionResult, linkedUsers *entities.LinkedUsers) error
       Next() Notifier
       SetNextNotifier(notifier Notifier)
   }
   ```

2. **Конкретные обработчики**:
   - `TelegramNotifier` - отправка через Telegram Bot API
   - `MailNotifier` - отправка через SMTP

#### Преимущества использования:

1. **Гибкость**: Легко добавлять новые способы уведомлений
2. **Надежность**: Если один канал недоступен, используется следующий
3. **Разделение ответственности**: Каждый нотификатор отвечает только за свой канал
4. **Конфигурируемость**: Порядок обработчиков настраивается при инициализации

#### Алгоритм работы:

1. MessageService получает результат сессии
2. Вызывает первый нотификатор в цепочке (обычно TelegramNotifier)
3. Нотификатор проверяет наличие контактов для своего канала
4. Если контакты найдены - пытается отправить сообщение
5. При успехе - возвращает успех
6. При неудаче или отсутствии контактов - передает управление следующему нотификатору
7. Процесс повторяется до успешной отправки или исчерпания цепочки

### Другие паттерны:

1. **Dependency Injection**: Внедрение зависимостей через конструкторы
2. **Repository Pattern**: Абстракция работы с внешними сервисами (AuthClient)
3. **Event-Driven Architecture**: Асинхронная обработка событий через NATS
4. **Template Method**: Общая структура обработки сообщений в нотификаторах

## Ключевые компоненты

### Entities (Сущности)
- **SessionResult**: Результат тестирования пользователя
- **LinkedUsers**: Связанные пользователи (студент и получатель уведомлений)
- **User**: Пользователь системы с контактной информацией

### Use Cases (Случаи использования)
- **MessageService**: Основной сервис обработки сообщений
- **Notifier Chain**: Цепочка нотификаторов для отправки уведомлений

### Adapters (Адаптеры)
- **TelegramNotifier**: Адаптер для отправки через Telegram
- **MailNotifier**: Адаптер для отправки через Email
- **AuthService**: Адаптер для получения данных пользователей
- **NATSConsumer**: Адаптер для получения сообщений из NATS

### Ports (Порты)
- **NATS Consumer**: Потребитель сообщений из брокера
- **MessageService Interface**: Интерфейс сервиса сообщений

## Безопасность и Надежность

1. **Graceful Degradation**: При недоступности одного канала используется другой
2. **Error Handling**: Обработка ошибок на каждом уровне цепочки
3. **Logging**: Подробное логирование процесса доставки
4. **Authentication**: Использование токенов для внешних API
5. **Validation**: Проверка входных данных на всех этапах

## Конфигурация

Сервис настраивается через переменные окружения:
- NATS connection settings
- Telegram Bot Token
- SMTP server settings
- Auth Service endpoint

## Мониторинг

Сервис предоставляет логирование для отслеживания:
- Получения сообщений из NATS
- Успешности отправки уведомлений
- Переключений между нотификаторами
- Ошибок в цепочке обработки

Такая архитектура обеспечивает высокую надежность доставки уведомлений и простоту расширения функциональности сервиса.

Необходимо пересмотреть архитекуру сервиса Notificationhub. Почему?
Текущая архитектура подразумевает сложную логику: 
восстановление события из шины; запрос связанных пользователей; обогащение события; цепочка отправки

нужно исключить пункт с походами в другие сервисы, то есть сервис должен работать только как порт и производить отправку на клиенты.
такой подход позволит минимизировать логику обработки события условно: нужно, чтобы мы обрабатывали событие, как срез байт и получателя и все.

то есть, большую часть логики надо перенести на reporting сервис, пусть он ходит по другим сервисам и собирает нужную инфу, потом, пережимает ее в события и отправляет на notify
