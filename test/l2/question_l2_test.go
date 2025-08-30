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

	"github.com/google/uuid"
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
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()

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
	t.Parallel()

	adminJWT := getJwt(t)

	userID, jwt := createUser(t, adminJWT, "Student")

	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("%s/%s/start_session", baseURL, userID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err := client.Do(req)
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
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err = client.Do(req)
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()

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

func TestCompleteSessionTwiceFailure(t *testing.T) {
	t.Parallel()

	adminJWT := getJwt(t)
	userID, jwt := createUser(t, adminJWT, "Student")

	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("%s/%s/start_session", baseURL, userID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err := client.Do(req)
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
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err = client.Do(req)
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()

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

	requestBody = map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	jsonBody, err = json.Marshal(requestBody)
	require.NoError(t, err)

	url = fmt.Sprintf("%s/%s/start_session", baseURL, userID)
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUserDeletion(t *testing.T) {
	adminJWT := getJwt(t)
	userID, _ := createUser(t, adminJWT, "Student")

	deleteUser(t, adminJWT, userID)
}

func TestErrorCases(t *testing.T) {
	client := &http.Client{Timeout: timeout}

	jwt := getJwt(t)
	// start session with not existings topics
	t.Run("NonExistentTopics", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"topics": []string{"not existning topic"},
		}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		userID, studentJWT := createUser(t, jwt, "Student")

		url := fmt.Sprintf("%s/%s/start_session", baseURL, userID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
		require.NoError(t, err)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer func() {
			err = resp.Body.Close()
			require.NoError(t, err)
		}()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// invalid type of topic field in json
	t.Run("InvalidRequestFormat", func(t *testing.T) {
		invalidJSON := `{"topics": "not an array"}`

		userID, studentJWT := createUser(t, jwt, "Student")
		url := fmt.Sprintf("%s/%s/start_session", baseURL, userID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(invalidJSON)))
		require.NoError(t, err)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer func() {
			err = resp.Body.Close()
			require.NoError(t, err)
		}()

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
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		userID, studentJWT := createUser(t, jwt, "Student")
		sessionID := fmt.Sprintf("%d", time.Now().UnixMilli())

		url := fmt.Sprintf("%s/%s/%s/complete_session", baseURL, userID, sessionID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(jsonBody)))
		require.NoError(t, err)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer func() {
			err = resp.Body.Close()
			require.NoError(t, err)
		}()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestConcurrentRequests(t *testing.T) {
	client := &http.Client{Timeout: timeout}
	jwt := getJwt(t)

	t.Run("ConcurrentTopicsRequests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req, err := http.NewRequest(http.MethodGet, baseURL+"/topics", nil)
				require.NoError(t, err)
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
				req.Header.Add("Content-Type", "application/json")

				resp, err := client.Do(req)
				require.NoError(t, err)
				if err != nil {
					results <- err
					return
				}
				defer func() {
					err = resp.Body.Close()
					require.NoError(t, err)
				}()

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

func TestUpdateUser(t *testing.T) {
	rootAdminJWT := getJwt(t)
	adminUserID, adminJWT := createUser(t, rootAdminJWT, "Admin")

	t.Run("AdminCanCreateUser", func(t *testing.T) {
		testUserID, _ := createUser(t, adminJWT, "Student")
		deleteUser(t, adminJWT, testUserID)
	})

	t.Run("AdminCanDeleteUser", func(t *testing.T) {
		testUserID, _ := createUser(t, rootAdminJWT, "Student")
		deleteUser(t, adminJWT, testUserID)
	})

	newLogin := "updated_student_" + uuid.New().String()
	newPassword := "new_student_password"
	updateUserToStudentWithCredentials(t, rootAdminJWT, adminUserID, newLogin, newPassword)
	studentJWT := getNewUserJwt(t, newLogin, newPassword)
	t.Run("StudentCannotCreateUser", func(t *testing.T) {
		client := &http.Client{Timeout: timeout}

		bodyDTO := &UserDTO{
			Username: "test_user_" + uuid.New().String(),
			Password: "test_password",
			Rights:   []string{"student"},
			Contacts: map[string]string{"email": "test@test.com"},
		}

		data, err := json.Marshal(&bodyDTO)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, "http://localhost:8090/auth/v1/add-user", bytes.NewReader(data))
		require.NoError(t, err)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("StudentCannotDeleteUser", func(t *testing.T) {
		client := &http.Client{Timeout: timeout}

		testUserID, _ := createUser(t, rootAdminJWT, "Student")

		req, err := http.NewRequest(http.MethodDelete, "http://localhost:8090/auth/v1/delete-user/"+testUserID, nil)
		require.NoError(t, err)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)

		deleteUser(t, rootAdminJWT, testUserID)
	})

	deleteUser(t, rootAdminJWT, adminUserID)
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

	data, err := json.Marshal(&AuthData{Login: "admin@kvs.ru", Password: "password123"})
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

func getNewUserJwt(t *testing.T, login, pass string) string {
	t.Helper()

	client := &http.Client{Timeout: timeout}
	type AuthData struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	data, err := json.Marshal(&AuthData{Login: login, Password: pass})
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

func createUser(t *testing.T, adminJWT string, userStatus string) (string, string) {
	t.Helper()

	var rights []string
	switch userStatus {
	case "Admin":
		rights = []string{"admin", "add_user", "delete_user", "view_topic_list", "start_session",
			"complete_session", "view_completed_sessions"}
	case "Mentor":
		rights = []string{"mentor", "view_topic_list", "start_session", "complete_session",
			"view_completed_sessions"}
	default:
		rights = []string{"student", "view_topic_list", "start_session", "complete_session"}
	}

	bodyDTO := &UserDTO{
		Username: uuid.NewString(),
		Password: uuid.NewString(),
		FullName: uuid.NewString(),
		Rights:   rights,
		Contacts: map[string]string{"phone": uuid.NewString(), "telegram": uuid.NewString()},
	}

	client := &http.Client{Timeout: timeout}
	data, err := json.Marshal(&bodyDTO)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "http://localhost:8090/auth/v1/add-user", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		closeErr := resp.Body.Close()
		require.NoError(t, closeErr)
	}()

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	type AddUserResponseDTO struct {
		// required: true
		UserID string `json:"user_id"`
	}

	var AddUserRespDTO AddUserResponseDTO
	err = json.NewDecoder(resp.Body).Decode(&AddUserRespDTO)
	require.NoError(t, err)

	require.NotEqual(t, "", AddUserRespDTO.UserID)

	return AddUserRespDTO.UserID, getNewUserJwt(t, bodyDTO.Username, bodyDTO.Password)
}

func deleteUser(t *testing.T, adminJWT string, userID string) {
	t.Helper()

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(http.MethodDelete, "http://localhost:8090/auth/v1/delete-user/"+userID, nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() {
		closeErr := resp.Body.Close()
		require.NoError(t, closeErr)
	}()

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func updateUserToStudentWithCredentials(t *testing.T, adminJWT string, userID string, newLogin string, newPassword string) {
	t.Helper()

	updateDTO := &UserDTO{
		Username: newLogin,
		Password: newPassword,
		Rights:   []string{"student", "view_topic_list", "start_session", "complete_session"},
	}

	client := &http.Client{Timeout: timeout}
	data, err := json.Marshal(&updateDTO)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPatch, "http://localhost:8090/auth/v1/update-user/"+userID, bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCompleteStudentWorkflowWithLinkedID(t *testing.T) {
	t.Parallel()

	adminJWT := getJwt(t)
	require.NotEmpty(t, adminJWT)

	mentorID, mentorJWT := createMentorWithEmail(t, adminJWT, "nvmaslenko@gmail.com")
	require.NotEmpty(t, mentorID)
	require.NotEmpty(t, mentorJWT)

	groupID := createGroup(t, adminJWT, "Test Group Email"+uuid.NewString(), mentorID)
	require.NotEmpty(t, groupID)

	studentID, studentJWT := createStudentWithGroup(t, adminJWT, groupID)
	require.NotEmpty(t, studentID)
	require.NotEmpty(t, studentJWT)

	sessionRequestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	sessionJSON, err := json.Marshal(sessionRequestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	sessionURL := fmt.Sprintf("%s/%s/start_session", baseURL, studentID)
	sessionReq, err := http.NewRequest(http.MethodPost, sessionURL, bytes.NewReader(sessionJSON))
	require.NoError(t, err)

	sessionReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
	sessionReq.Header.Add("Content-Type", "application/json")

	sessionResp, err := client.Do(sessionReq)
	require.NoError(t, err)
	defer sessionResp.Body.Close()

	require.Equal(t, http.StatusCreated, sessionResp.StatusCode)

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

	sessionBody, err := io.ReadAll(sessionResp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(sessionBody, &sessionResponse)
	require.NoError(t, err)

	require.NotEmpty(t, sessionResponse.SessionID)
	require.Greater(t, len(sessionResponse.Questions), 0)
	require.Equal(t, 2, len(sessionResponse.Topics))

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

	completeJSON, err := json.Marshal(completeBody)
	require.NoError(t, err)

	completeURL := fmt.Sprintf("%s/%s/%s/complete_session", baseURL, studentID, sessionResponse.SessionID)
	completeReq, err := http.NewRequest(http.MethodPost, completeURL, bytes.NewReader(completeJSON))
	require.NoError(t, err)

	completeReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
	completeReq.Header.Add("Content-Type", "application/json")

	completeResp, err := client.Do(completeReq)
	require.NoError(t, err)
	defer completeResp.Body.Close()

	require.Equal(t, http.StatusOK, completeResp.StatusCode)

	completeBodyResult, err := io.ReadAll(completeResp.Body)
	require.NoError(t, err)

	var resultResponse struct {
		IsSuccess bool   `json:"is_success"`
		Grade     string `json:"grade"`
	}
	err = json.Unmarshal(completeBodyResult, &resultResponse)
	require.NoError(t, err)

	require.NotEmpty(t, resultResponse.Grade)
}

func createGroup(t *testing.T, adminJWT string, title string, mentorID string) string {
	t.Helper()

	type AddGroupRequestDTO struct {
		Title    string `json:"title"`
		LinkedID string `json:"linked_id"`
	}

	groupDTO := &AddGroupRequestDTO{
		Title:    title,
		LinkedID: mentorID,
	}

	client := &http.Client{Timeout: timeout}
	data, err := json.Marshal(groupDTO)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "http://localhost:8090/auth/v1/add-group", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	type AddGroupResponseDTO struct {
		GroupID string `json:"group_id"`
	}

	var addGroupRespDTO AddGroupResponseDTO
	err = json.NewDecoder(resp.Body).Decode(&addGroupRespDTO)
	require.NoError(t, err)

	require.NotEmpty(t, addGroupRespDTO.GroupID)

	return addGroupRespDTO.GroupID
}

func createStudentWithGroup(t *testing.T, adminJWT string, groupID string) (string, string) {
	t.Helper()

	rights := []string{"student", "view_topic_list", "start_session", "complete_session"}

	studentUsername := uuid.NewString()
	studentPassword := uuid.NewString()
	studentFullName := "Student " + uuid.NewString()

	bodyDTO := &UserDTO{
		Username: studentUsername,
		Password: studentPassword,
		FullName: studentFullName,
		Rights:   rights,
		Contacts: map[string]string{
			"email":    studentUsername + "@student.test.com",
			"phone":    uuid.NewString(),
			"telegram": uuid.NewString(),
		},
		GroupID: groupID,
	}

	client := &http.Client{Timeout: timeout}
	data, err := json.Marshal(&bodyDTO)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "http://localhost:8090/auth/v1/add-user", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		closeErr := resp.Body.Close()
		require.NoError(t, closeErr)
	}()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	type AddUserResponseDTO struct {
		UserID string `json:"user_id"`
	}

	var addUserRespDTO AddUserResponseDTO
	err = json.NewDecoder(resp.Body).Decode(&addUserRespDTO)
	require.NoError(t, err)

	require.NotEmpty(t, addUserRespDTO.UserID)

	studentJWT := getNewUserJwt(t, studentUsername, studentPassword)

	return addUserRespDTO.UserID, studentJWT
}

func createMentorWithEmail(t *testing.T, adminJWT string, email string) (string, string) {
	t.Helper()

	rights := []string{"mentor", "view_topic_list", "start_session", "complete_session", "view_completed_sessions"}

	mentorUsername := uuid.NewString()
	mentorPassword := uuid.NewString()
	mentorFullName := uuid.NewString()

	bodyDTO := &UserDTO{
		Username: mentorUsername,
		Password: mentorPassword,
		FullName: mentorFullName,
		Rights:   rights,
		Contacts: map[string]string{
			"email":    email,
			"phone":    uuid.NewString(),
			"telegram": uuid.NewString(),
		},
	}

	client := &http.Client{Timeout: timeout}
	data, err := json.Marshal(&bodyDTO)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "http://localhost:8090/auth/v1/add-user", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		closeErr := resp.Body.Close()
		require.NoError(t, closeErr)
	}()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	type AddUserResponseDTO struct {
		UserID string `json:"user_id"`
	}

	var addUserRespDTO AddUserResponseDTO
	err = json.NewDecoder(resp.Body).Decode(&addUserRespDTO)
	require.NoError(t, err)

	require.NotEmpty(t, addUserRespDTO.UserID)

	mentorJWT := getNewUserJwt(t, mentorUsername, mentorPassword)

	return addUserRespDTO.UserID, mentorJWT
}

func TestCompleteStudentWorkflowWithTelegramLinkedID(t *testing.T) {
	t.Parallel()

	adminJWT := getJwt(t)
	require.NotEmpty(t, adminJWT)

	mentorID, mentorJWT := createMentorWithTelegram(t, adminJWT, "164718531")
	require.NotEmpty(t, mentorID)
	require.NotEmpty(t, mentorJWT)

	groupID := createGroup(t, adminJWT, "Test Group Telegram", mentorID)
	require.NotEmpty(t, groupID)

	studentID, studentJWT := createStudentWithGroup(t, adminJWT, groupID)
	require.NotEmpty(t, studentID)
	require.NotEmpty(t, studentJWT)

	sessionRequestBody := map[string]interface{}{
		"topics": []string{"Базы данных", "Базовые типы в Go"},
	}

	sessionJSON, err := json.Marshal(sessionRequestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	sessionURL := fmt.Sprintf("%s/%s/start_session", baseURL, studentID)
	sessionReq, err := http.NewRequest(http.MethodPost, sessionURL, bytes.NewReader(sessionJSON))
	require.NoError(t, err)

	sessionReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
	sessionReq.Header.Add("Content-Type", "application/json")

	sessionResp, err := client.Do(sessionReq)
	require.NoError(t, err)
	defer sessionResp.Body.Close()

	require.Equal(t, http.StatusCreated, sessionResp.StatusCode)

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

	sessionBody, err := io.ReadAll(sessionResp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(sessionBody, &sessionResponse)
	require.NoError(t, err)

	require.NotEmpty(t, sessionResponse.SessionID)
	require.Greater(t, len(sessionResponse.Questions), 0)
	require.Equal(t, 2, len(sessionResponse.Topics))

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

	completeJSON, err := json.Marshal(completeBody)
	require.NoError(t, err)

	completeURL := fmt.Sprintf("%s/%s/%s/complete_session", baseURL, studentID, sessionResponse.SessionID)
	completeReq, err := http.NewRequest(http.MethodPost, completeURL, bytes.NewReader(completeJSON))
	require.NoError(t, err)

	completeReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", studentJWT))
	completeReq.Header.Add("Content-Type", "application/json")

	completeResp, err := client.Do(completeReq)
	require.NoError(t, err)
	defer completeResp.Body.Close()

	require.Equal(t, http.StatusOK, completeResp.StatusCode)

	completeBodyResult, err := io.ReadAll(completeResp.Body)
	require.NoError(t, err)

	var resultResponse struct {
		IsSuccess bool   `json:"is_success"`
		Grade     string `json:"grade"`
	}
	err = json.Unmarshal(completeBodyResult, &resultResponse)
	require.NoError(t, err)

	require.NotEmpty(t, resultResponse.Grade)
}

func createMentorWithTelegram(t *testing.T, adminJWT string, telegramID string) (string, string) {
	t.Helper()

	rights := []string{"mentor", "view_topic_list", "start_session", "complete_session", "view_completed_sessions"}

	mentorUsername := uuid.NewString()
	mentorPassword := uuid.NewString()
	mentorFullName := uuid.NewString()

	bodyDTO := &UserDTO{
		Username: mentorUsername,
		Password: mentorPassword,
		FullName: mentorFullName,
		Rights:   rights,
		Contacts: map[string]string{
			"telegram": telegramID,
			"phone":    uuid.NewString(),
			"email":    uuid.NewString() + "@test.com",
		},
	}

	client := &http.Client{Timeout: timeout}
	data, err := json.Marshal(&bodyDTO)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "http://localhost:8090/auth/v1/add-user", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminJWT))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		closeErr := resp.Body.Close()
		require.NoError(t, closeErr)
	}()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	type AddUserResponseDTO struct {
		UserID string `json:"user_id"`
	}

	var addUserRespDTO AddUserResponseDTO
	err = json.NewDecoder(resp.Body).Decode(&addUserRespDTO)
	require.NoError(t, err)

	require.NotEmpty(t, addUserRespDTO.UserID)

	mentorJWT := getNewUserJwt(t, mentorUsername, mentorPassword)

	return addUserRespDTO.UserID, mentorJWT
}

type UserDTO struct {
	// required: true
	Username string `json:"name"`
	// required: true
	Password string `json:"password"`
	// required: true
	FullName string `json:"fullname"`
	// required: true
	Rights   []string          `json:"rights"`
	Contacts map[string]string `json:"contacts,omitempty"`
	GroupID  string            `json:"group_id,omitempty"`
}

func TestCompleteSessionWithExpiredTime(t *testing.T) {
	t.Skip("this test skipped becouse time_to_respond" +
		" in deployment file of question service has value > 1s")
	t.Parallel()

	adminJWT := getJwt(t)

	userID, jwt := createUser(t, adminJWT, "Student")

	requestBody := map[string]interface{}{
		"topics": []string{"Базы данных"},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("%s/%s/start_session", baseURL, userID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err := client.Do(req)
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

	time.Sleep(11 * time.Second)
	jsonBody, err = json.Marshal(completeBody)
	require.NoError(t, err)
	url = fmt.Sprintf("%s/%s/%s/complete_session", baseURL, userID, sessionResponse.SessionID)
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Content-Type", "application/json")

	require.NoError(t, err)

	resp, err = client.Do(req)
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	var resultResponse struct {
		IsSuccess bool   `json:"is_success"`
		Grade     string `json:"grade"`
	}
	err = json.Unmarshal(body, &resultResponse)
	require.NoError(t, err)

	require.Contains(t, resultResponse.Grade, "session expired")
	require.NotEmpty(t, resultResponse.Grade)
}
