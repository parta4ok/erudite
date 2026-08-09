# Архитектура Question service

## Введение
Question service (далее - сервис) является основой бизнес логики проекта Knowledge Validation System (далее - kvs, erudite).
Сервис позволяет пользователям:
- Запрашивать список тем;
- Создавать сессию для заданных тем;
- Загружать ответы пользователя;
- Вычислять результаты и формировать заключение о том, сдал ли студент сессию или нет;

## System Design
Сервис не предполагает интенсивного использования, и вопросы, связанные с распределением нагрузки и оптимизации, будут рассматриваться в порядке их поступления. 
Сервис написан на языке Go. 
В качестве точки входа используется публичный HTTP REST API.
Для проверки правомерности операций используются JWT и интроспекция на Auth service (далее - iam) сервисе. 
В качестве хранилища используется СУБД PostgreSQL. 
Результаты сессии публикуются посредством брокера NATS в стрим с сессиями.
### Структура сервиса
```mermaid
graph TD
    subgraph Client[User]
        A[request]
    end

    subgraph Service[Go Service]
        B[HTTP REST API Public Port]
        C[Auth client]
        D[Cases]
        E[NATS Publisher]
    end

    subgraph DB[PostgreSQL]
        F[(DataBase)]
    end

    subgraph IAM[Auth Service IAM]
        G[introspect JWT]
    end

    subgraph NATS[NATS Broker]
        H[(Stream session result)]
    end

    A --> B
    B --> |1| C
    C --> G
    B --> |2| D 
    D --> F
    D --> E
    E --> |session result| H
```
## Структура проекта
### Диаграмма классов
```mermaid
classDiagram
    SingleSelectionQuestion --|> Question
    MultiSelectionQuestion --|> Question
    TrueOrFalseSelectionQuestion --|> Question

	QuestionFactory ..> Question: create

    Session --o SessionState
    Session --|> StateHolder
    Session --o IDGenerator
    InitSessionState --|> SessionState
    ActiveSessionState --|> SessionState
    CompletedSessionState --|> SessionState

    InitSessionState --o SessionStorage
    InitSessionState --o StateHolder
    ActiveSessionState --o StateHolder
    ActiveSessionState --o Question
    CompletedSessionState --o UserAnswer
    CompletedSessionState --o Question
    Session ..> SessionResult: create

    SessionServiceBase --o Storage
    SessionServiceBase --o SessionStorage
    SessionServiceBase --o IDGenerator
	SessionServiceBase --|> SessionService

	SessionServiceBusDecorator --o MessageBroker
	SessionServiceBusDecorator --o SessionService
	SessionServiceBusDecorator --|> SessionService

	Nats --|> MessageBroker
	
	Postgres --|> Storage
	Postgres --|> SessionStorage
	Postgres --o QuestionFactory

	Uint64Generator --|> IDGenerator

	PublicServer --o SessionService
	PublicServer --o AuthClient
	PublicServer --o Accessor

    namespace entities {
        class Question{
            <<interface>>
            +ID() string
	        +Type() QuestionType
	        +Topic() string
	        +Subject() string
	        +Variants() []string
	        +IsAnswerCorrect(ans *UserAnswer) bool
        }
        class SingleSelectionQuestion{
            -id            string
	        -topic         string
	        -subject       string
	        -variants      []string
	        -correctAnswer string
            
            +ID() string
	        +Type() QuestionType
	        +Topic() string
	        +Subject() string
	        +Variants() []string
	        +IsAnswerCorrect(ans *UserAnswer) bool
        }

        class MultiSelectionQuestion{
            -id            string
	        -topic         string
	        -subject       string
	        -variants      []string
	        -correctAnswer []string
            
            +ID() string
	        +Type() QuestionType
	        +Topic() string
	        +Subject() string
	        +Variants() []string
	        +IsAnswerCorrect(ans *UserAnswer) bool
        }

        class TrueOrFalseSelectionQuestion {
            -id            string
	        -topic         string
	        -subject       string
	        -correctAnswer bool
            
            +ID() string
	        +Type() QuestionType
	        +Topic() string
	        +Subject() string
	        +Variants() []string
	        +IsAnswerCorrect(ans *UserAnswer) bool
        }

		class QuestionFactory{
			+NewQuestion(id string, questionType QuestionType, topic string, subject string, variants []string, correctAnswer []string) (Question, error)
		}

        class Session {
	        -userID    string
	        -sessionID string
	        -topics    []string
	        -state SessionState

            +ChangeState(state SessionState)
            +GetQuestions() ([]Question, error)
            +GetSessionID() string
            +GetSessionDurationLimit() (time.Duration, error)
            +GetSessionResult() (*SessionResult, error)
            +GetStartedAt() (time.Time, error)
            +GetStatus() string
            +GetTopics() []string
            +GetUserAnswers() ([]*UserAnswer, error)
            +GetUserID() string
            +IsDailySessionLimitReached() bool
            +SetQuestions(qestions map[string]Question, duration time.Duration) error
            +SetUserAnswer(answers []*UserAnswer) error
        }

        class SessionState{
            <<interface>>
	        GetStatus() string
	        GetQuestions() ([]Question, error)
	        GetStartedAt() (time.Time, error)
	        GetUserAnswers() ([]*UserAnswer, error)
	        SetQuestions(qestions map[string]Question, duration time.Duration) error
	        SetUserAnswer(answers []*UserAnswer) error
	        GetSessionResult() (*SessionResult, error)
	        GetSessionDurationLimit() (time.Duration, error)
	        IsExpired() (bool, error)
	        IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
        }

        class StateHolder {
            <<interface>>
	        ChangeState(state SessionState)
        }

        class InitSessionState {
	        -stateHolder    StateHolder
	        -sessionStorage SessionStorage
            
            GetStatus() string
	        GetQuestions() ([]Question, error)
	        GetStartedAt() (time.Time, error)
	        GetUserAnswers() ([]*UserAnswer, error)
	        SetQuestions(qestions map[string]Question, duration time.Duration) error
	        SetUserAnswer(answers []*UserAnswer) error
	        GetSessionResult() (*SessionResult, error)
	        GetSessionDurationLimit() (time.Duration, error)
	        IsExpired() (bool, error)
	        IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
        }

        class ActiveSessionState {
	        -holder    StateHolder
	        -questions map[string]Question
	        -startedAt time.Time
	        -duration  time.Duration
            
            GetStatus() string
	        GetQuestions() ([]Question, error)
	        GetStartedAt() (time.Time, error)
	        GetUserAnswers() ([]*UserAnswer, error)
	        SetQuestions(qestions map[string]Question, duration time.Duration) error
	        SetUserAnswer(answers []*UserAnswer) error
	        GetSessionResult() (*SessionResult, error)
	        GetSessionDurationLimit() (time.Duration, error)
	        IsExpired() (bool, error)
	        IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
        }

        class CompletedSessionState {
	        -questions map[string]Question
	        -answers   []*UserAnswer
	        -holder    StateHolder
	        -startedAt time.Time
	        -isExpired bool
            
            GetStatus() string
	        GetQuestions() ([]Question, error)
	        GetStartedAt() (time.Time, error)
	        GetUserAnswers() ([]*UserAnswer, error)
	        SetQuestions(qestions map[string]Question, duration time.Duration) error
	        SetUserAnswer(answers []*UserAnswer) error
	        GetSessionResult() (*SessionResult, error)
	        GetSessionDurationLimit() (time.Duration, error)
	        IsExpired() (bool, error)
	        IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
        }

        class IDGenerator {
            <<interface>>
	        GenerateID() string
        }

        class SessionStorage {
            <<interface>>
	        IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
        }

        class UserAnswer {
	        -questionID string
	        -answer     []string

            +GetQuestionID() string
            +GetSelections() []string
        }

        class SessionResult{
	        UserID      string
	        Topics      []string
	        Questions   map[string][]string
	        UserAnswers map[string][]string
	        IsExpire    bool
	        IsSuccess   bool
	        Grade       string
        }
    }

    namespace cases {
        class Storage {
            <<interface>>
	        +GetTopics(ctx context.Context) ([]string, error)
	        +GetQuestions(ctx context.Context, topics []string) ([]entities.Question, error)
	        +StoreSession(ctx context.Context, session *entities.Session) error
	        +GetSessionBySessionID(ctx context.Context, sessionID string) (*entities.Session, error)
	        +GetAllCompletedUserSessions(ctx context.Context, userID string) ([]*entities.Session, error)
        }

        class MessageBroker {
            <<interface>>
	        +SessionFinishedEvent(ctx context.Context, sessionResult *entities.SessionResult) error
        }

		class SessionService {
			<<interface>>
			+CompleteSession(ctx context.Context, sessionID string, answers []*entities.UserAnswer) (*entities.SessionResult, error)
			+CreateSession(ctx context.Context, userID string, topics []string) (string, map[string]entities.Question, error)
			+GetAllCompletedUserSessions(ctx context.Context, userID string) ([]*entities.Session, error)
			+ShowTopics(ctx context.Context) ([]string, error)
		}

        class SessionServiceBase {
	        -storage        Storage
	        -sessionStorage entities.SessionStorage
	        -generator      entities.IDGenerator
	        -topicDuration  time.Duration
            +CompleteSession(ctx context.Context, sessionID string, answers []*entities.UserAnswer) (*entities.SessionResult, error)
            +CreateSession(ctx context.Context, userID string, topics []string) (string, map[string]entities.Question, error)
            +GetAllCompletedUserSessions(ctx context.Context, userID string) ([]*entities.Session, error)
            +ShowTopics(ctx context.Context) ([]string, error)
        }

		class SessionServiceBusDecorator {
			-sessionService SessionService
			-messageBroker  MessageBroker
			-timeoutEvent   time.Duration
			+CompleteSession(ctx context.Context, sessionID string, answers []*entities.UserAnswer) (*entities.SessionResult, error)
            +CreateSession(ctx context.Context, userID string, topics []string) (string, map[string]entities.Question, error)
            +GetAllCompletedUserSessions(ctx context.Context, userID string) ([]*entities.Session, error)
            +ShowTopics(ctx context.Context) ([]string, error)		
		}
    }

	namespace adapters {
		class Nats {
			-conn nats.Connection
			-stram string
			+SessionFinishedEvent(ctx context.Context, sessionResult *entities.SessionResult) error
		}

		class Postgres {
			-conn             postgres.Connection
			-questionFactory *entities.QuestionFactory

			+GetAllCompletedUserSessions(ctx context.Context, userID string) ([]*entities.Session, error)
			+GetQuestions(ctx context.Context, topics []string) ([]entities.Question, error)
			+GetSessionBySessionID(ctx context.Context, sessionID string) (*entities.Session, error)
			+GetTopics(ctx context.Context) ([]string, error)
			+IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
			+StoreSession(ctx context.Context, session *entities.Session) error
		}

		class Uint64Generator {
			+GenerateID() string
		}

		class AuthClient {
			-client *client.AuthClient
			+Introspect(ctx context.Context, jwt string) (*entities.Claims, error)
		}
	}

	namespace ports {
		class PublicServer{
			-router       *chi.Mux
			-server       *http.Server
			-service      Service
			-introspector Introspector
			-accessor     Accessor
			+CompleteSession(resp http.ResponseWriter, req *http.Request)
			+GetAllCompletedUserSessions(resp http.ResponseWriter, req *http.Request)
			+GetTopics(resp http.ResponseWriter, req *http.Request)
			+StartSession(resp http.ResponseWriter, req *http.Request)
		}
		class Accessor{
			<<interface>>
			+HasPermission(ctx context.Context, rights []string) (bool, error)
		}
	}
```

### Диаграммы последовательностей
#### Диаграмма запроса списка тем
```mermaid
sequenceDiagram
    actor S as Student
    participant P as PublicPort
    participant AU as AuthService
    participant AC as Accessor
    participant SS as SessionService
    participant PG as Storage

    S ->> P: GetTopicList
    activate P
    P ->> AU: Introspect(JWT)
    activate AU
    AU -->> P: claims
    deactivate AU
    P ->> AC: HasPermission(claims, rights []string)
    activate AC
    alt not enough right
        AC -->> P: user has not
        P -->> S: forbidden
    end
    AC --> P: ok
    deactivate AC
    P ->> SS: ShowTopics()
    activate SS
    SS ->> PG: GetTopics()
    activate PG
    PG -->> SS: topic list
    deactivate PG
    SS -->> P: topic list
    deactivate SS
    P --> S: topic list
    deactivate P
```
#### Диаграмма создания сессии
```mermaid
sequenceDiagram
    actor S as Student
    participant P as PublicPort
    participant AU as AuthService
    participant AC as Accessor
    participant SS as SessionService
    participant PG as Storage

    S ->> P: StartSession
    activate P
    P ->> AU: Introspect(JWT)
    activate AU
    AU -->> P: claims
    deactivate AU
    P ->> AC: HasPermission(claims, rights []string)
    activate AC
    alt not enough right
        AC -->> P: user has not
        P -->> S: forbidden
    end
    AC --> P: ok
    deactivate AC
    P ->> SS: CreateSession(userID, topics)
    activate SS
    create participant Session 
    SS ->> Session: NewSession(userID, topics, generator, sessionStorage)
    activate Session
    Session ->> PG: IsDailySessionLimit(userID, topics)
    activate PG
    alt user already try to make session today
    PG -->> Session: true
    Session -->> SS: true
    SS -->> P: ErrForbidden
    P -->> S: ErrForbidden
    end
    PG -->> Session: false
    deactivate PG
    Session -->> SS: false
    deactivate Session
    SS ->> PG: GetQuestions(topics)
    PG -->> SS: Questions
    SS ->> SS: generate QuestionsMap
    SS ->> Session: SetQuestions(QuestionsMap)
    activate Session
    Session -->> SS: ok
    deactivate Session
    
    SS ->> PG: StoreSession(Session)
    activate PG
    PG -->> SS: ok
    deactivate PG
    SS ->> Session: GetSessionID()
    activate Session
    Session -->> SS: ID
    deactivate Session
    SS -->> P: ID, QuestionMap
    deactivate SS
    P -->> S: ID, QuestionMap
    deactivate P
```

#### Диаграмма завершения сессии
```mermaid
sequenceDiagram
    actor S as Student
    participant P as PublicPort
    participant AU as AuthService
    participant AC as Accessor
    participant SSBD as SessionServiceBusDecorator
    participant SS as SessionService
    participant PG as Storage

    S ->> P: CompleteSession
    activate P
    P ->> AU: Introspect(JWT)
    activate AU
    AU -->> P: claims
    deactivate AU
    P ->> AC: HasPermission(claims, rights []string)
    activate AC
    alt not enough right
        AC -->> P: user has not
        P -->> S: forbidden
    end
    AC --> P: ok
    deactivate AC

    P ->> SSBD: CompleteSession(sessionID, UserAnswers)
    activate SSBD
    SSBD ->> SS: CompleteSession(sessionID, UserAnswers)
    activate SS
    SS ->> PG: GetSessionBySessionID(sessionID)
    activate PG
    create participant Session
    PG ->> Session: NewInitSessionState()
    deactivate PG
    Session ->> Session: restore session state
    activate Session
    Session -->> SS: session
    deactivate Session
    SS ->> Session: SetUserAnswers(UserAnswers)
    activate Session
    Session -->> SS: ok
    deactivate Session
    SS ->> Session: GetSessionResult()
    activate Session
    Session -->> SS: SessionResult
    deactivate Session
    SS ->> PG: StoreSession(session)
    activate PG
    PG -->> SS: ok
    deactivate PG
    SS -->> SSBD: SessionResult
    deactivate SS
    participant MB as MessageBroker
    SSBD ->> MB: go SessionFinishedEvent(SessionResult)
    SSBD -->> P: SessionResult
    deactivate SSBD
    P --> S: SessionResult
    deactivate P
```
#### Диаграмма загрузки всех сессий студента
```mermaid
sequenceDiagram
    actor M as Mentor
    participant P as PublicPort
    participant AU as AuthService
    participant AC as Accessor
    participant SS as SessionService
    participant PG as Storage

    M ->> P: GetAllCompletedUserSessions
    activate P
    P ->> AU: Introspect(JWT)
    activate AU
    AU -->> P: claims
    deactivate AU
    P ->> AC: HasPermission(claims, rights []string)
    activate AC
    alt not enough right
        AC -->> P: user has not
        P -->> M: forbidden
    end
    AC --> P: ok
    deactivate AC
    P ->> SS: GetAllCompletedUserSessions(userID)
    activate SS
    SS ->> PG: GetAllCompletedUserSessions(userID)
    activate PG
    PG -->> SS: list of user sessions by current day
    deactivate PG
    SS -->> P: list of user sessions by current day
    deactivate SS
    P -->> M: list of user sessions by current day
    deactivate P
```
