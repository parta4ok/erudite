package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/parta4ok/kvs/toolkit/pkg/port"
	"github.com/pkg/errors"
)

var (
	ErrSheduler = errors.New("sheduler error")
)

var (
	_ port.BasePort = (*Sheduler)(nil)
)

const (
	croneJobType = "cron_job"
)

type Sheduler struct {
	shdlr gocron.Scheduler
}

func NewSheduler() (*Sheduler, error) {
	sheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, errors.Wrapf(ErrSheduler, "new sheduler failure: %v", err)
	}
	return &Sheduler{
		shdlr: sheduler,
	}, nil
}

func (s *Sheduler) NewJob(interval time.Duration, job any, args ...any) error {
	slog.Info("NewJob started")
	_, err := s.shdlr.NewJob(gocron.DurationJob(interval), gocron.NewTask(job, args...))
	if err != nil {
		slog.Error("new job creation failed", "error", err)
		return errors.Wrapf(ErrSheduler, "new job creation failed: %v", err)
	}

	slog.Info("NewJob successfully completed")
	return nil
}

func (s *Sheduler) Start(ctx context.Context) error {
	slog.Info("Sheduler started")
	s.shdlr.Start()

	<-ctx.Done()
	slog.Info("Sheduler completed")
	return nil
}

func (s *Sheduler) Stop(ctx context.Context) error {
	slog.Info("Sheduler stop started")
	done := make(chan error, 1)

	go func() {
		done <- s.shdlr.Shutdown()
	}()

	select {
	case err := <-done:
		if err != nil {
			slog.Error("shudown failure", "error", err)
			return errors.Wrapf(ErrSheduler, "shutdown with err: %v", err)
		}
		return nil
	case <-ctx.Done():
		slog.Error("shudown by context", "error", ctx.Err())
		return ctx.Err()
	}
}

func (s *Sheduler) Type() string {
	return croneJobType
}
