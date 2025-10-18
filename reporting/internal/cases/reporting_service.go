package cases

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/pkg/errors"
)

var (
	ErrReportingServiceStopped = errors.New(
		"service has stopped and the operation cannot be completed")
)

type ReportingService struct {
	broker         MessageBroker
	representers   []Representer
	authClient     AuthClient
	questionClient QuestionClient
	tasks          []func() error
	errChan        chan error
	workersLimit   int
	cond           sync.Cond
	stopSignal     *atomic.Bool
	wg             sync.WaitGroup
}

func NewReportingService(
	broker MessageBroker,
	representers []Representer,
	authClient AuthClient,
	questionClient QuestionClient,
	workersLimit int,
) (*ReportingService, error) {
	if broker == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "broker not set")
	}

	for _, representer := range representers {
		if representer == nil {
			return nil, errors.Wrap(entities.ErrInvalidParam, "one or more representer not set")
		}
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

	stop := &atomic.Bool{}
	stop.Store(false)

	service := &ReportingService{
		broker:         broker,
		representers:   representers,
		authClient:     authClient,
		questionClient: questionClient,
		workersLimit:   workersLimit,
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
		slog.Error(err.Error())
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

func (service *ReportingService) GetPassedTopicsByGroups(ctx context.Context, mentorID string,
	reportFormat string) error {
	if service.stopSignal.Load() {
		return errors.Wrap(ErrReportingServiceStopped, "GetPassedTopicsByGroups")
	}

	var selectedRepresenter Representer
	for _, representer := range service.representers {
		if representer.GetReportFormat() == reportFormat {
			selectedRepresenter = representer
		}
	}

	if selectedRepresenter == nil {
		return errors.Wrap(entities.ErrInvalidParam, "unknown report format or representer not seted")
	}

	fn := func() error {
		students, err := service.authClient.GetMentorGroups(ctx, mentorID)
		if err != nil {
			return errors.Wrap(err, "get mentor group with auth client failure")
		}

		studentsIDs := make([]string, 0, len(students))
		for _, student := range students {
			studentsIDs = append(studentsIDs, student.ID)
		}

		passedTopics, err := service.questionClient.GetPassedStudentsTopics(ctx, studentsIDs)
		if err != nil {
			return errors.Wrap(err, "get passed topics by students ids with question client failure")
		}

		for _, student := range students {
			student.PassedTopics = passedTopics[student.ID]
		}

		report, err := entities.NewPassedTopicsReport(students)
		if err != nil {
			return errors.Wrap(err, "creating new report about passed topics by groups")
		}

		if err = service.broker.ReportEvent(ctx, report, selectedRepresenter); err != nil {
			return errors.Wrap(err, "sending report event was failed")
		}

		return nil
	}

	service.cond.L.Lock()
	defer service.cond.L.Unlock()

	service.tasks = append(service.tasks, fn)
	service.cond.Signal()

	return nil
}

func (service *ReportingService) Stop() error {
	if service.stopSignal.Load() {
		return errors.Wrap(ErrReportingServiceStopped, "service already stopped")
	}

	service.stopSignal.Store(true)
	service.cond.Broadcast()
	service.wg.Wait()

	close(service.errChan)

	return nil
}
