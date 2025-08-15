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
	"github.com/parta4ok/kvs/question/pkg/dto"
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

	msgCh := make(chan dto.EventDTO, 1)
	nc, err := natsDriver.Connect(natsUrl)
	require.NoError(t, err)
	defer nc.Drain()

	sub, err := nc.Subscribe(subject, func(msg *natsDriver.Msg) {
		var messageDto dto.EventDTO
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
		require.Equal(t, finishedSession.UserID, recv.Payload.UserID)
		// Можно проверить другие поля по необходимости
	case <-ctx.Done():
		t.Errorf("message not recieved")
	}
}
