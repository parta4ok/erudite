//go:build KVS_TEST_L2

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	authBaseURL = "http://localhost:8090/auth/v1"
)

func TestDynamicRegistration(t *testing.T) {
	t.Parallel()

	type DynamicRegistrationDTO struct {
		UserID   string `json:"id"`
		Provider string `json:"provider"`
	}

	body := DynamicRegistrationDTO{
		UserID:   "nvmaslenko@yandex.com",
		Provider: "email",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodPost, authBaseURL+"/dynamic-registration", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDynamicRegistration_Failure_AlreadyExists(t *testing.T) {
	t.Parallel()

	type DynamicRegistrationDTO struct {
		UserID   string `json:"id"`
		Provider string `json:"provider"`
	}

	body := DynamicRegistrationDTO{
		UserID:   "mentor1@kvs.ru",
		Provider: "email",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodPost, authBaseURL+"/dynamic-registration", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}
