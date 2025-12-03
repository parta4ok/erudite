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
	"github.com/parta4ok/kvs/question/internal/entities/event"
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

func TestPublisher_Publish(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pub, err := publisher.NewPublisher(natsUrl)
	require.NoError(t, err)
	natsStream, err := nats.NewPublisher(pub, subject)
	require.NoError(t, err)

	msgCh := make(chan []byte, 1)
	nc, err := natsDriver.Connect(natsUrl)
	require.NoError(t, err)
	defer nc.Drain()

	sub, err := nc.Subscribe(subject, func(msg *natsDriver.Msg) {
		msgCh <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	sessionResultDTO := natsDTO.SessionResultDTO{
		UserID:      uuid.NewString(),
		Topics:      []string{uuid.NewString(), uuid.NewString()},
		Questions:   map[string][]string{uuid.NewString(): {uuid.NewString(), uuid.NewString()}},
		UserAnswers: map[string][]string{uuid.NewString(): {uuid.NewString(), uuid.NewString()}},
		IsExpire:    false,
		IsSuccess:   true,
		Grade:       "85%",
	}

	payload, err := json.Marshal(sessionResultDTO)
	require.NoError(t, err)

	sessionEvent, err := event.NewSessionCompleteEvent(payload)
	require.NoError(t, err)

	err = natsStream.Publish(ctx, sessionEvent)
	require.NoError(t, err)

	select {
	case recv := <-msgCh:
		var receivedDTO natsDTO.SessionResultDTO
		err := json.Unmarshal(recv, &receivedDTO)
		require.NoError(t, err)

		require.Equal(t, sessionResultDTO.UserID, receivedDTO.UserID)
		require.Equal(t, sessionResultDTO.Topics, receivedDTO.Topics)
		require.Equal(t, sessionResultDTO.Questions, receivedDTO.Questions)
		require.Equal(t, sessionResultDTO.UserAnswers, receivedDTO.UserAnswers)
		require.Equal(t, sessionResultDTO.IsExpire, receivedDTO.IsExpire)
		require.Equal(t, sessionResultDTO.IsSuccess, receivedDTO.IsSuccess)
		require.Equal(t, sessionResultDTO.Grade, receivedDTO.Grade)
	case <-ctx.Done():
		t.Errorf("message not received")
	}
}
