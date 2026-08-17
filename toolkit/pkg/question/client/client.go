package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	apiVersion       = "/kvs/v1"
	passedTopicsPath = "/passed_topics"
)

var (
	ErrInternal   = errors.New("internal error")
	ErrBadRequest = errors.New("bad request")
)

type Client struct {
	c        http.Client
	basePath string
}

func New(basePath string) (*Client, error) {
	return &Client{
		c:        http.Client{},
		basePath: basePath,
	}, nil
}

type StudentsTopics struct {
	StudentsTopics map[string][]Topic `json:"students_topics"`
}

type Topic struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type StudentsIDs struct {
	Students []string `json:"students_ids"`
}

func (client *Client) GetPassedStudentsTopics(ctx context.Context, students []string) (
	*StudentsTopics, error) {
	slog.Info("GetPassedStudentsTopics on toolkit client started", slog.Any("students", students))

	urlRaw, err := url.Parse(client.basePath + apiVersion + passedTopicsPath)
	if err != nil {
		slog.Error("url parse", "error", err.Error())
		return nil, errors.Wrapf(ErrInternal, "url parse failure: %v", err)
	}

	studentsIDs := StudentsIDs{
		Students: students,
	}

	data, err := json.Marshal(studentsIDs)
	if err != nil {
		slog.Error("json marshal", "error", err.Error())
		return nil, errors.Wrapf(ErrInternal, "marshal failure: %v", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		urlRaw.String(),
		bytes.NewReader(data),
	)
	if err != nil {
		slog.Error("NewRequestWithContext", "error", err.Error(), slog.String("url", urlRaw.String()))
		return nil, errors.Wrapf(ErrInternal, "new request failure: %v", err)
	}
	slog.Info("request data", slog.String("method", req.Method), slog.String("url", req.URL.String()))
	req.Header.Set("Content-Type", "application/json")

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	
	resp, err := client.c.Do(req)
	if err != nil {
		slog.Error("client.do failure", "error", err.Error())
		return nil, errors.Wrapf(ErrInternal, "client.do failure: %v", err)
	}

	defer resp.Body.Close() //nolint:errcheck // ok

	if resp.StatusCode != http.StatusOK {
		slog.Error("response status not equal status OK", "current status", resp.StatusCode)
		return nil, errors.Wrap(client.processErr(resp.StatusCode), "request failure")
	}

	var studentsTopics = StudentsTopics{
		StudentsTopics: make(map[string][]Topic),
	}

	if err := json.NewDecoder(resp.Body).Decode(&studentsTopics); err != nil {
		slog.Info("json.NewDecoder", "error", err.Error())
		return nil, errors.Wrapf(ErrInternal, "decode body failure: %v", err)
	}

	return &studentsTopics, nil
}

func (client *Client) processErr(statusCode int) error {
	switch statusCode { //nolint:gocritic // will be extension on the next steps
	case http.StatusBadRequest:
		return ErrBadRequest
	}
	return ErrInternal
}
