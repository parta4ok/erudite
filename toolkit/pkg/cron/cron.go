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
	ErrScheduler = errors.New("scheduler error")
)

var (
	_ port.BasePort = (*Scheduler)(nil)
)

const (
	croneJobType = "cron_job"
)

type Scheduler struct {
	shdlr gocron.Scheduler
}

func NewScheduler() (*Scheduler, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, errors.Wrapf(ErrScheduler, "new scheduler failure: %v", err)
	}
	return &Scheduler{
		shdlr: scheduler,
	}, nil
}

func (s *Scheduler) NewJob(interval time.Duration, job any, args ...any) error {
	slog.Info("NewJob started")
	_, err := s.shdlr.NewJob(gocron.DurationJob(interval), gocron.NewTask(job, args...))
	if err != nil {
		slog.Error("new job creation failed", "error", err)
		return errors.Wrapf(ErrScheduler, "new job creation failed: %v", err)
	}

	slog.Info("NewJob successfully completed")
	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	slog.Info("Scheduler started")
	s.shdlr.Start()

	<-ctx.Done()
	slog.Info("Scheduler completed")
	return nil
}

func (s *Scheduler) Stop(ctx context.Context) error {
	slog.Info("Scheduler stop started")
	done := make(chan error, 1)

	go func() {
		done <- s.shdlr.Shutdown()
	}()

	select {
	case err := <-done:
		if err != nil {
			slog.Error("shudown failure", "error", err)
			return errors.Wrapf(ErrScheduler, "shutdown with err: %v", err)
		}
		return nil
	case <-ctx.Done():
		slog.Error("shudown by context", "error", ctx.Err())
		return ctx.Err()
	}
}

func (s *Scheduler) Type() string {
	return croneJobType
}
