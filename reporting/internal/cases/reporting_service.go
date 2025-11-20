package cases

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

var (
	ErrReportingServiceStopped = errors.New(
		"service has stopped and the operation cannot be completed")
)

type ReportingService struct {
	broker         MessageBroker
	representer    entities.Representer
	format         entities.Format
	asyncTimeout   time.Duration
	authClient     AuthClient
	questionClient QuestionClient
	tasks          []func() error
	errChan        chan error
	workersLimit   int
	cond           sync.Cond
	stopSignal     *atomic.Bool
	wg             sync.WaitGroup
}

//nolint:funlen //ok
func NewReportingService(
	broker MessageBroker,
	representer entities.Representer,
	format string,
	authClient AuthClient,
	questionClient QuestionClient,
	workersLimit int,
	asyncTimeout time.Duration,
) (*ReportingService, error) {
	if broker == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "broker not set")
	}

	if representer == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "representer not set")
	}

	if format == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "format not set")
	}

	if authClient == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "auth client not set")
	}

	if questionClient == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "question client not set")
	}

	if workersLimit == 0 {
		return nil, errors.Wrap(entities.ErrInvalidParam, "workers limit must be greater than 0")
	}

	if asyncTimeout < 3*time.Second {
		asyncTimeout = 3 * time.Second
	}

	stop := &atomic.Bool{}
	stop.Store(false)

	service := &ReportingService{
		broker:         broker,
		representer:    representer,
		format:         entities.Format(format),
		authClient:     authClient,
		questionClient: questionClient,
		workersLimit:   workersLimit,
		asyncTimeout:   asyncTimeout,
		tasks:          make([]func() error, 0),
		errChan:        make(chan error, workersLimit),
		cond:           *sync.NewCond(&sync.Mutex{}),
		stopSignal:     stop,
		wg:             sync.WaitGroup{},
	}

	for range service.workersLimit {
		service.wg.Add(1)
		go service.Worker()
	}

	go service.errorHandlerStart()

	return service, nil
}

func (service *ReportingService) errorHandlerStart() {
	for err := range service.errChan {
		slog.Error("error message", "error", err)
	}
}

func (service *ReportingService) Worker() {
	for {
		service.cond.L.Lock()

		for len(service.tasks) == 0 && !service.stopSignal.Load() {
			service.cond.Wait()
		}

		if service.stopSignal.Load() {
			service.cond.L.Unlock()
			break
		}

		task := service.tasks[0]
		if len(service.tasks) == 1 {
			service.tasks = make([]func() error, 0)
		}
		if len(service.tasks) > 1 {
			service.tasks = service.tasks[1:]
		}

		service.cond.L.Unlock()

		if err := task(); err != nil {
			service.errChan <- errors.Wrap(err, "worker processing was end with failure")
		}
	}

	service.wg.Done()
}

//nolint:funlen //ok
func (service *ReportingService) GetPassedTopicsByGroups(ctx context.Context, mentorID string,
) error {
	_, span, cancel := tracing.GlobalTracer().Start(ctx, "GetPassedTopicsByGroups")
	defer cancel()

	if service.stopSignal.Load() {
		err := ErrReportingServiceStopped
		span.SetError(err, "GetPassedTopicsByGroups")
		slog.Error("GetPassedTopicsByGroups", "error", err)
		return errors.Wrap(err, "GetPassedTopicsByGroups")
	}

	fn := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), service.asyncTimeout)
		defer cancel()

		students, err := service.authClient.GetMentorGroups(ctx, mentorID)
		if err != nil {
			slog.Error("GetMentorGroups", "error", err)
			return errors.Wrap(err, "get mentor group with auth client failure")
		}

		studentsIDs := make([]string, 0, len(students))
		for _, student := range students {
			studentsIDs = append(studentsIDs, student.ID)
		}

		passedTopics, err := service.questionClient.GetPassedStudentsTopics(ctx, studentsIDs)
		if err != nil {
			slog.Error("get passed topics by students ids with question client failure", "error", err)
			return errors.Wrap(err, "get passed topics by students ids with question client failure")
		}

		for _, student := range students {
			student.PassedTopics = passedTopics[student.ID]
		}

		recipient, err := service.authClient.GetUserByID(ctx, mentorID)
		if err != nil {
			slog.Error("get user by id with auth client failure", "error", err)
			return errors.Wrap(err, "get user by id with auth client failure")
		}

		var report entities.Report
		passedTopicsReport, err := entities.NewPassedTopicsReport(students)
		if err != nil {
			slog.Error("creating new report about passed topics by groups", "error", err)
			return errors.Wrap(err, "creating new report about passed topics by groups")
		}

		report = passedTopicsReport

		return service.eventProcessing(ctx, report, recipient)
	}

	service.cond.L.Lock()
	defer service.cond.L.Unlock()

	service.tasks = append(service.tasks, fn)
	service.cond.Signal()

	return nil
}

func (service *ReportingService) DeliverySessionResult(
	ctx context.Context,
	session *entities.SessionResult,
) error {
	_, span, cancel := tracing.GlobalTracer().Start(ctx, "DeliverySessionResult")
	defer cancel()

	if service.stopSignal.Load() {
		err := ErrReportingServiceStopped
		span.SetError(err, "DeliverySessionResult")
		slog.Error("DeliverySessionResult", "error", err)
		return errors.Wrap(err, "DeliverySessionResult")
	}

	fn := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), service.asyncTimeout)
		defer cancel()

		linkedUsers, err := service.authClient.GetLinkedUsers(ctx, session.UserID)
		if err != nil {
			slog.Error("get linked users with auth client failure", "error", err)
			return errors.Wrap(err, "get linked users with auth client failure")
		}

		recipient := linkedUsers.Mentor

		var report entities.Report

		sessionResultReport, err := entities.NewSessionResult(
			linkedUsers.Student.ID,
			linkedUsers.Student.Fullname,
			linkedUsers.Student.GroupID,
			session.Topics,
			session.Questions,
			session.UserAnswer,
			session.IsExpire,
			session.IsSuccess,
			session.Resume,
		)
		if err != nil {
			slog.Error("creature new session result failure", "error", err)
			return errors.Wrap(err, "creature new session result failure")
		}
		if sessionResultReport == nil {
			err := entities.ErrInternal
			slog.Error("sessionResultReport creature was failed", "error", err)
			return errors.Wrap(err, "sessionResultReport creature was failed")
		}

		report = sessionResultReport

		return service.eventProcessing(ctx, report, recipient)
	}

	service.cond.L.Lock()
	defer service.cond.L.Unlock()

	service.tasks = append(service.tasks, fn)
	service.cond.Signal()

	return nil
}

func (service *ReportingService) Stop() error {
	if service.stopSignal.Load() {
		err := ErrReportingServiceStopped
		slog.Error("service already stopped", "error", err)
		return errors.Wrap(err, "service already stopped")
	}

	service.stopSignal.Store(true)
	service.cond.Broadcast()
	service.wg.Wait()

	close(service.errChan)

	return nil
}

func (service *ReportingService) eventProcessing(ctx context.Context,
	report entities.Report,
	recipient *entities.User) error {
	report.SetMessageType()

	message, err := service.representer.CovertToFormat(service.format, report)
	if err != nil {
		slog.Error(fmt.Sprintf("convert to format %s was failed", service.format), "error", err)
		return errors.Wrapf(err, "convert to format %s was failed", service.format)
	}

	concreteEvent, err := entities.NewBaseEvent(report.Kind(), service.format, message, recipient)
	if err != nil {
		slog.Error("new base event creature was failed", "error", err)
		return errors.Wrap(err, "new base event creature was failed")
	}

	if err = service.broker.ReportEvent(ctx, concreteEvent); err != nil {
		slog.Error("sending report event was failed", "error", err)
		return errors.Wrap(err, "sending report event was failed")
	}

	return nil
}
