package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const (
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
	urlRaw, err := url.Parse(client.basePath + passedTopicsPath)
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "url parse failure: %v", err)
	}

	studentsIDs := StudentsIDs{
		Students: students,
	}

	data, err := json.Marshal(studentsIDs)
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "marshal failure: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlRaw.String(), bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "new request failure: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.c.Do(req)
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "client.do failure: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(client.processErr(resp.StatusCode), "request failure")
	}

	var studentsTopics = StudentsTopics{
		StudentsTopics: make(map[string][]Topic),
	}

	if err := json.NewDecoder(resp.Body).Decode(&studentsTopics); err != nil {
		return nil, errors.Wrapf(ErrInternal, "decode body failure: %v", err)
	}

	return &studentsTopics, nil
}

func (client *Client) processErr(statusCode int) error {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrBadRequest

	}
	return ErrInternal
}
