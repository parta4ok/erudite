# Диаграммы Knowledge Validation Service

Этот документ содержит все архитектурные диаграммы системы, созданные с помощью Mermaid.

## 📊 Диаграмма классов

Показывает структуру основных сущностей системы и их взаимосвязи:

```mermaid
classDiagram
    %% Entities Layer
    class Session {
        -uint64 userID
        -uint64 sessionID
        -[]string topics
        -SessionState state
        +GetSessionID() uint64
        +GetUserID() uint64
        +GetTopics() []string
        +SetQuestions(map[uint64]Question, time.Duration) error
        +SetUserAnswer([]*UserAnswer) error
        +GetStatus() string
        +GetSessionResult() (*SessionResult, error)
        +ChangeState(SessionState)
    }

    class SessionState {
        <<interface>>
        +GetStatus() string
        +GetQuestions() ([]Question, error)
        +SetQuestions(map[uint64]Question, time.Duration) error
        +SetUserAnswer([]*UserAnswer) error
        +GetSessionResult() (*SessionResult, error)
        +IsExpired() (bool, error)
    }

    class Question {
        <<interface>>
        +ID() uint64
        +Type() QuestionType
        +Topic() string
        +Subject() string
        +Variants() []string
        +IsAnswerCorrect(*UserAnswer) bool
    }

    class SingleSelectionQuestion {
        -uint64 id
        -string topic
        -string subject
        -[]string variants
        -string correctAnswer
        +ID() uint64
        +Type() QuestionType
        +IsAnswerCorrect(*UserAnswer) bool
    }

    class SessionService {
        -Storage storage
        -SessionStorage sessionStorage
        -IDGenerator generator
        -time.Duration topicDuration
        +ShowTopics(context.Context) ([]string, error)
        +CreateSession(context.Context, uint64, []string) (uint64, map[uint64]Question, error)
        +CompleteSession(context.Context, uint64, []UserAnswer) (*SessionResult, error)
    }

    %% Relationships
    Session --> SessionState : uses
    Question <|-- SingleSelectionQuestion
    SessionService --> Session : creates
```

## 🔄 Диаграмма последовательности - Создание сессии

Показывает поток создания новой тестовой сессии:

```mermaid
sequenceDiagram
    participant Client
    participant Server
    participant SessionService
    participant Storage
    participant Session
    participant InitState
    participant ActiveState

    Client->>+Server: POST /kvs/v1/{user_id}/start_session
    Server->>+SessionService: CreateSession(ctx, userID, topics)
    SessionService->>+Storage: GetQuestions(ctx, topics)
    Storage-->>-SessionService: []Question
    SessionService->>+Session: NewSession(userID, topics, generator, storage)
    Session->>+InitState: NewInitSessionState(session, storage)
    InitState-->>-Session: InitState instance
    Session-->>-SessionService: Session instance
    SessionService->>+Session: SetQuestions(questionsMap, duration)
    Session->>+InitState: SetQuestions(questionsMap, duration)
    InitState->>+ActiveState: NewActiveSessionState(questions, holder, duration)
    ActiveState-->>-InitState: ActiveState instance
    InitState-->>-Session: nil (success)
    Session-->>-SessionService: nil (success)
    SessionService-->>-Server: sessionID, questionsMap, nil
    Server-->>-Client: 201 Created + SessionDTO
```

## 🏁 Диаграмма последовательности - Завершение сессии

Показывает поток завершения сессии и подсчета результатов:

```mermaid
sequenceDiagram
    participant Client
    participant Server
    participant SessionService
    participant Storage
    participant Session
    participant ActiveState
    participant CompletedState

    Client->>+Server: POST /kvs/v1/{user_id}/{session_id}/complete_session
    Server->>+SessionService: CompleteSession(ctx, sessionID, userAnswers)
    SessionService->>+Storage: GetSessionBySessionID(ctx, sessionID)
    Storage-->>-SessionService: Session instance
    SessionService->>+Session: SetUserAnswer(userAnswers)
    Session->>+ActiveState: SetUserAnswer(userAnswers)
    ActiveState->>+CompletedState: NewCompletedSessionState(questions, holder, answers, isExpired)
    CompletedState-->>-ActiveState: CompletedState instance
    ActiveState-->>-Session: nil (success)
    Session-->>-SessionService: nil (success)
    SessionService->>+Session: GetSessionResult()
    Session->>+CompletedState: GetSessionResult()
    
    alt Session not expired
        CompletedState->>CompletedState: Calculate results
        CompletedState-->>Session: SessionResult{IsSuccess, Grade}
    else Session expired
        CompletedState-->>Session: SessionResult{false, "session expired"}
    end
    
    Session-->>-SessionService: SessionResult
    SessionService-->>-Server: SessionResult, nil
    Server-->>-Client: 200 OK + SessionResultDTO
```

## 🏗️ Диаграмма архитектуры системы

Показывает общую архитектуру и взаимодействие компонентов:

```mermaid
graph TB
    %% External
    Client[Client Application]
    DB[(PostgreSQL Database)]
    
    %% HTTP Layer
    subgraph "HTTP Layer (Port)"
        Server[HTTP Server<br/>Chi Router]
        Handlers[HTTP Handlers<br/>GetTopics<br/>StartSession<br/>CompleteSession]
    end
    
    %% Application Layer
    subgraph "Application Layer (Use Cases)"
        ServiceInterface[Service Interface]
        SessionService[Session Service<br/>Business Logic]
        StorageInterface[Storage Interface]
    end
    
    %% Domain Layer
    subgraph "Domain Layer (Entities)"
        Session[Session<br/>State Machine]
        States[Session States<br/>Init → Active → Completed]
        Questions[Question Types<br/>Single/Multi/TrueOrFalse]
        UserAnswer[User Answer]
        SessionResult[Session Result]
    end
    
    %% Infrastructure Layer
    subgraph "Infrastructure Layer (Adapters)"
        PostgresAdapter[PostgreSQL Adapter<br/>Database Operations]
        IDGenerator[ID Generator<br/>Unique IDs]
    end
    
    %% DTOs
    subgraph "Data Transfer Objects"
        DTOs[SessionDTO<br/>SessionResultDTO<br/>UserAnswersListDTO<br/>ErrorDTO]
    end
    
    %% Connections
    Client --> Server
    Server --> Handlers
    Handlers --> ServiceInterface
    ServiceInterface --> SessionService
    SessionService --> StorageInterface
    StorageInterface --> PostgresAdapter
    PostgresAdapter --> DB
    
    SessionService --> Session
    Session --> States
    Session --> Questions
    Session --> UserAnswer
    Session --> SessionResult
    
    SessionService --> IDGenerator
    
    Handlers --> DTOs
    DTOs --> Handlers
```

## 🔄 State Machine диаграмма

Показывает жизненный цикл сессии и переходы между состояниями:

```mermaid
stateDiagram-v2
    [*] --> InitState : NewSession()
    
    InitState --> ActiveState : SetQuestions()
    InitState --> [*] : Error/Timeout
    
    ActiveState --> CompletedState : SetUserAnswer()
    ActiveState --> CompletedState : Timeout/Expired
    ActiveState --> [*] : Error
    
    CompletedState --> [*] : GetSessionResult()
    
    note right of InitState
        - Session created
        - Waiting for questions
        - Can set questions
        - Cannot get results
    end note
    
    note right of ActiveState
        - Questions loaded
        - Timer started
        - Can accept answers
        - Cannot get results yet
    end note
    
    note right of CompletedState
        - Answers submitted
        - Results calculated
        - Can get session result
        - Cannot modify answers
    end note
```

## 🗄️ Диаграмма базы данных

Показывает структуру базы данных и связи между таблицами:

```mermaid
erDiagram
    TOPICS {
        int id PK
        int topic_id UK
        varchar name UK
    }
    
    QUESTION_TYPES {
        int id PK
        varchar name UK
    }
    
    QUESTIONS {
        bigint id PK
        int question_id UK
        int question_type_id FK
        int topic_id FK
        varchar subject
        text_array variants
        text_array correct_answers
        int usage_count
    }
    
    SESSIONS {
        int id PK
        bigint session_id
        int user_id
        varchar state
        text_array topics
        int_array questions
        json answers
        timestamp created_at
        bigint duration_limit
        boolean is_expired
        boolean is_passed
        varchar comment
        timestamp updated_at
    }
    
    TOPICS ||--o{ QUESTIONS : "has"
    QUESTION_TYPES ||--o{ QUESTIONS : "defines"
    SESSIONS ||--o{ QUESTIONS : "contains"
```

## 📊 Диаграмма потоков данных

Показывает, как данные перемещаются через систему:

```mermaid
flowchart TD
    A[HTTP Request] --> B[Server Handler]
    B --> C[DTO Validation]
    C --> D[Service Layer]
    D --> E[Business Logic]
    E --> F[Storage Interface]
    F --> G[PostgreSQL Adapter]
    G --> H[(Database)]
    
    H --> I[Query Results]
    I --> J[Entity Creation]
    J --> K[Business Processing]
    K --> L[DTO Conversion]
    L --> M[HTTP Response]
    
    subgraph "Error Handling"
        N[Error Occurred]
        N --> O[Error Wrapping]
        O --> P[Error DTO]
        P --> Q[HTTP Error Response]
    end
    
    D -.-> N
    E -.-> N
    F -.-> N
    G -.-> N
```

## 🔧 Как использовать диаграммы

1. **Для понимания архитектуры** - начните с диаграммы компонентов
2. **Для изучения потоков** - используйте диаграммы последовательности
3. **Для понимания данных** - изучите диаграмму классов и ER-диаграмму
4. **Для отладки состояний** - обратитесь к State Machine

## 📝 Обновление диаграмм

При изменении архитектуры обновляйте соответствующие диаграммы:

1. Отредактируйте Mermaid код в этом файле
2. Проверьте корректность синтаксиса
3. Обновите документацию при необходимости

Диаграммы можно просматривать в любом редакторе с поддержкой Mermaid или на [mermaid.live](https://mermaid.live).
