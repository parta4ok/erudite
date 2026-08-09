package telegram

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
)

var (
	_ cases.Notifier = (*TelegramNotifier)(nil)
)

var (
	tgTitles = [...]entities.DeliveryService{
		entities.DeliveryService("tg"),
		entities.DeliveryService("telegram"),
		entities.DeliveryService("тг"),
		entities.DeliveryService("телеграм"),
		entities.DeliveryService("телега"),
	}
)

type TelegramNotifier struct {
	bot  *tgbotapi.BotAPI
	next cases.Notifier
}

func NewTelegramNotifier(next cases.Notifier, token string) (*TelegramNotifier, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &TelegramNotifier{
		bot:  bot,
		next: next,
	}, nil
}

func (tg *TelegramNotifier) Notify(ctx context.Context, message entities.Event) error {
	ctx, span, cancel := tracer.Start(ctx, "TelegramNotifySpan")
	defer cancel()

	slog.Info("Notify for telegram notifier started")

	recipientID := tg.checkTelegramInContacts(message.GetRecipient().Contacts)
	if recipientID == entities.Contact("") {
		slog.Warn("Recipient telegram address not found")
		if nextNotifier := tg.Next(); nextNotifier != nil {
			return nextNotifier.Notify(ctx, message)
		}
		slog.Warn("Telegram notifier is last. Message not be sent")
		return nil
	}

	id, err := strconv.ParseInt(recipientID.String(), 10, 64)
	if err != nil {
		slog.Warn("Recipient telegram address incorrect")
		if nextNotifier := tg.Next(); nextNotifier != nil {
			return nextNotifier.Notify(ctx, message)
		}
		slog.Warn("Telegram notifier is last. Message not be sent")
		return nil
	}

	fileName := fmt.Sprintf("%s.%s", message.Kind(), message.Format())
	fileReader := bytes.NewReader(message.Payload())

	document := tgbotapi.NewDocument(id, tgbotapi.FileReader{
		Name:   fileName,
		Reader: fileReader,
	})

	_, err = tg.bot.Send(document)
	if err != nil {
		span.SetError(err)
		slog.Warn("Recipient telegram send document failure:" + err.Error())
		if nextNotifier := tg.Next(); nextNotifier != nil {
			return nextNotifier.Notify(ctx, message)
		}
		slog.Warn("Telegram notifier is last. Document not be sent")
		return nil
	}

	return nil
}

func (tg *TelegramNotifier) SetNextNotifier(notifier cases.Notifier) {
	slog.Info("Setting in telegram notifier next notifier")
	tg.next = notifier
}

func (tg *TelegramNotifier) Next() cases.Notifier {
	if tg.next == nil {
		slog.Info("No next notifier set for telegram notifier")
		return nil
	}
	return tg.next
}

func (tg *TelegramNotifier) checkTelegramInContacts(
	contacts map[entities.DeliveryService]entities.Contact) entities.Contact {
	slog.Info("Checking telegram in contacts started")

	for _, probableContact := range tgTitles {
		address, ok := contacts[probableContact]
		if ok {
			slog.Info("Checking telegram in contacts finished, telegram address found")
			return address
		}
	}

	slog.Info("Checking telegram in contacts finished, contacts not found")
	return ""
}
