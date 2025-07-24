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

const (
	rootUserID = "1"
)

func Test_Topics_Success(t *testing.T) {
	t.Parallel()

	token := getJwt(t)
	require.NotEqual(t, "", token)

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/topics", nil)
	require.NoError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
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

func Test_Topics_Unauthorized(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/topics", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCreateSession(t *testing.T) {
	t.Parallel()

	userID := rootUserID

	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	token := getJwt(t)
	require.NotEqual(t, "", token)

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/start_session", baseURL, userID), bytes.NewBuffer(jsonBody))
	require.NoError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var response struct {
		SessionID string        `json:"session_id"`
		Topics    []string      `json:"topics"`
		Questions []interface{} `json:"questions"`
	}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	require.NotEqual(t, response.SessionID, 0)

	require.NotEqual(t, len(response.Questions), 0)
}

func TestCompleteSession(t *testing.T) {
	t.Skip()
	t.Parallel()

	userID := fmt.Sprintf("%d", time.Now().UnixMicro())
	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("%s/%s/start_session", baseURL, userID)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	var sessionResponse struct {
		SessionID string `json:"session_id"`
		Questions []struct {
			ID           string   `json:"question_id"`
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

	require.NotEqual(t, sessionResponse.SessionID, "")

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

	url = fmt.Sprintf("%s/%s/%s/complete_session", baseURL, userID, sessionResponse.SessionID)
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

		userID := fmt.Sprintf("%d", time.Now().UnixMilli())
		sessionID := fmt.Sprintf("%d", time.Now().UnixMilli())
		urlSuff := fmt.Sprintf("%s/%s/complete_session", userID, sessionID)

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
	QuestionID string   `json:"question_id"`
	Answers    []string `json:"answers"`
}

type UserAnswersListDTO struct {
	AnswersList []UserAnswerDTO `json:"user_answers"`
}

func getJwt(t *testing.T) string {
	t.Helper()

	client := &http.Client{Timeout: timeout}
	type AuthData struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	data, err := json.Marshal(&AuthData{Login: "admin", Password: "password123"})
	require.NoError(t, err)

	resp, err := client.Post("http://localhost:8090/auth/v1/signin", "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	defer resp.Body.Close()

	type Token struct {
		Token string `json:"token"`
	}

	var token Token

	err = json.NewDecoder(resp.Body).Decode(&token)
	require.NoError(t, err)

	return token.Token
}
