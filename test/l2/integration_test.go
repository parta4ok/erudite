//go:build KVS_TEST_L2

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080/kvs/v1"
	timeout = 30 * time.Second
)

func TestTopicsEndpoint(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(baseURL + "/topics")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response struct {
		Topics []string `json:"topics"`
	}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	require.Greater(t, len(response.Topics), 0)
}

func TestCreateSession(t *testing.T) {
	t.Parallel()

	userID := uint64(12345)

	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("%s/%d/start_session", baseURL, userID)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var response struct {
		SessionID uint64        `json:"session_id"`
		Topics    []string      `json:"topics"`
		Questions []interface{} `json:"questions"`
	}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	require.NotEqual(t, response.SessionID, 0)

	require.NotEqual(t, len(response.Questions), 0)
}

func TestCompleteSession(t *testing.T) {
	t.Parallel()

	userID := time.Now().UnixMicro()
	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("%s/%d/start_session", baseURL, userID)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	var sessionResponse struct {
		SessionID uint64 `json:"session_id"`
		Questions []struct {
			ID           uint64   `json:"question_id"`
			QuestionType string   `json:"question_type"`
			Topic        string   `json:"topic"`
			Subject      string   `json:"subject"`
			Variants     []string `json:"variants"`
		} `json:"questions"`
		Topics []string `json:"topics"`
	}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &sessionResponse)
	require.NoError(t, err)

	require.Greater(t, sessionResponse.SessionID, uint64(0))

	var answers []UserAnswerDTO
	for _, question := range sessionResponse.Questions {
		answers = append(answers, UserAnswerDTO{
			QuestionID: question.ID,
			Answers:    question.Variants[:1],
		})
	}

	completeBody := UserAnswersListDTO{
		AnswersList: answers,
	}

	jsonBody, err = json.Marshal(completeBody)
	require.NoError(t, err)

	url = fmt.Sprintf("%s/%d/%d/complete_session", baseURL, userID, sessionResponse.SessionID)
	resp, err = client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	var resultResponse struct {
		IsSuccess bool   `json:"is_success"`
		Grade     string `json:"grade"`
	}
	err = json.Unmarshal(body, &resultResponse)
	require.NoError(t, err)

	require.NotEmpty(t, resultResponse.Grade)
}

func TestErrorCases(t *testing.T) {
	client := &http.Client{Timeout: timeout}

	// start session with not existings topics
	t.Run("NonExistentTopics", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"topics": []string{"not existning topic"},
		}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		resp, err := client.Post(baseURL+"/1234/start_session", "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// invalid type of topic field in json
	t.Run("InvalidRequestFormat", func(t *testing.T) {
		invalidJSON := `{"topics": "not an array"}`

		resp, err := client.Post(baseURL+"/123/start_session", "application/json", bytes.NewReader([]byte(invalidJSON)))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// completed of not existings session
	t.Run("NonExistentSession", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"user_answer": []map[string]interface{}{
				{
					"question_id": 1,
					"answers":     []string{"test"},
				},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)

		userID := time.Now().UnixMilli()
		sessionID := time.Now().UnixMilli()
		urlSuff := fmt.Sprintf("%d/%d/complete_session", userID, sessionID)

		resp, err := client.Post(baseURL+urlSuff, "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestConcurrentRequests(t *testing.T) {
	client := &http.Client{Timeout: timeout}

	t.Run("ConcurrentTopicsRequests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := client.Get(baseURL + "/topics")
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
					return
				}

				results <- nil
			}()
		}

		for i := 0; i < numRequests; i++ {
			err := <-results
			require.NoError(t, err)
		}
	})
}

type UserAnswerDTO struct {
	QuestionID uint64   `json:"question_id"`
	Answers    []string `json:"answers"`
}

type UserAnswersListDTO struct {
	AnswersList []UserAnswerDTO `json:"user_answers"`
}
