//go:build KVS_TEST_L1

package nats_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	natsDriver "github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/question/internal/adapter/message_broker/nats"
	"github.com/parta4ok/kvs/question/internal/entities"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/stretchr/testify/require"
)

const (
	subject = "sessions.result"
)

var (
	natsUrl = os.Getenv("TEST_NATS_CONN")
)

func TestPublisher_SessionFinishedEvent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pub, err := publisher.NewPublisher(natsUrl)
	require.NoError(t, err)
	natsStream, err := nats.NewPublisher(pub, subject)
	require.NoError(t, err)

	msgCh := make(chan natsDTO.EventDTO, 1)
	nc, err := natsDriver.Connect(natsUrl)
	require.NoError(t, err)
	defer nc.Drain()

	sub, err := nc.Subscribe(subject, func(msg *natsDriver.Msg) {
		var messageDto natsDTO.EventDTO
		err := json.Unmarshal(msg.Data, &messageDto)
		require.NoError(t, err)
		msgCh <- messageDto
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	finishedSession := &entities.SessionResult{
		UserID:      uuid.NewString(),
		Topics:      []string{uuid.NewString(), uuid.NewString()},
		Questions:   map[string][]string{uuid.NewString(): {uuid.NewString(), uuid.NewString()}},
		UserAnswers: map[string][]string{uuid.NewString(): {uuid.NewString(), uuid.NewString()}},
		IsExpire:    false,
		IsSuccess:   false,
		Grade:       "10%",
	}

	err = natsStream.SessionFinishedEvent(ctx, finishedSession)
	require.NoError(t, err)

	select {
	case recv := <-msgCh:
		require.Equal(t, nats.SessionFinishedEventType, recv.EventType)

		var sessionResultDTO natsDTO.SessionResultDTO
		err := json.Unmarshal(recv.Payload, &sessionResultDTO)
		require.NoError(t, err)

		require.Equal(t, finishedSession.UserID, sessionResultDTO.UserID)
		require.Equal(t, finishedSession.Topics, sessionResultDTO.Topics)
		require.Equal(t, finishedSession.Questions, sessionResultDTO.Questions)
		require.Equal(t, finishedSession.UserAnswers, sessionResultDTO.UserAnswers)
		require.Equal(t, finishedSession.IsExpire, sessionResultDTO.IsExpire)
		require.Equal(t, finishedSession.IsSuccess, sessionResultDTO.IsSuccess)
		require.Equal(t, finishedSession.Grade, sessionResultDTO.Grade)
	case <-ctx.Done():
		t.Errorf("message not received")
	}
}
