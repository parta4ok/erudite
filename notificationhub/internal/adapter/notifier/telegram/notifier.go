package telegram

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
)

var (
	_ cases.Notifier = (*TelegramNotifier)(nil)
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

func (tg *TelegramNotifier) Notify(sessionResult *entities.SessionResult,
	recipient *entities.Recipient) error {
	slog.Info("Notify for telegram notifier started")

	recipientID := tg.checkTelegramInContacts(recipient)
	if recipientID == "" {
		slog.Warn("Recipient telegram address not found")
		if nextNotifier := tg.Next(); nextNotifier != nil {
			return nextNotifier.Notify(sessionResult, recipient)
		}
		slog.Warn("Telegram notifier is last. Message not be sent")
		return nil
	}

	id, err := strconv.ParseInt(recipientID, 10, 64)
	if err != nil {
		slog.Warn("Recipient telegram address incorrect")
		if nextNotifier := tg.Next(); nextNotifier != nil {
			return nextNotifier.Notify(sessionResult, recipient)
		}
		slog.Warn("Telegram notifier is last. Message not be sent")
		return nil
	}

	message := tg.generateMessage(sessionResult)

	msg := tgbotapi.NewMessage(id, message)
	_, err = tg.bot.Send(msg)
	if err != nil {
		slog.Warn("Recipient telegram send message failure:" + err.Error())
		if nextNotifier := tg.Next(); nextNotifier != nil {
			return nextNotifier.Notify(sessionResult, recipient)
		}
		slog.Warn("Telegram notifier is last. Message not be sent")
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

func (tg *TelegramNotifier) generateMessage(sessionResult *entities.SessionResult) string {
	message := fmt.Sprintf("userID: %s\n\ntopics: %s\n\nresult: %s\n\nsuccess: %t\n\n",
		sessionResult.GetUserID(), strings.Join(sessionResult.Topics, "; "), sessionResult.Resume,
		sessionResult.IsSuccess)

	var resultStr string
	for question, answers := range sessionResult.UserAnswer {
		answersJoined := strings.Join(answers, ";")
		resultStr += fmt.Sprintf("Вопрос: %s.\nОтвет пользователя: %s.\nВарианты ответов: %s.\n",
			question, answersJoined, strings.Join(sessionResult.Questions[question], "; "))
		resultStr += "\n----"
	}

	message += resultStr

	return message
}

func (tg *TelegramNotifier) checkTelegramInContacts(recipient *entities.Recipient) string {
	slog.Info("Checking mail in contacts started")

	contacts := []string{"tg", "telegram", "тг", "телеграм"}

	for _, probableContact := range contacts {
		recipientTelegramAddress, ok := recipient.Contacts[probableContact]
		if ok {
			slog.Info("Checking telegram in contacts finished, telegram found")
			return recipientTelegramAddress
		}
	}

	slog.Info("Checking telegram in contacts finished, contacts not found")
	return ""
}
