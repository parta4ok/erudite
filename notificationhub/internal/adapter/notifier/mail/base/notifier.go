package base

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/pkg/errors"
)

const (
	TitlePrefix = "Результаты тестирования для студента: %s"
)

type MailNotifier struct {
	next cases.Notifier

	host     string
	baseMail string
	basePort string
	pwd      string
}

func NewMailNotifier(
	next cases.Notifier,
	host, baseMail, basePort, pwd string) (*MailNotifier, error) {
	mailNotifier := &MailNotifier{}

	if host == "" {
		return mailNotifier.processErr("host")
	}

	if baseMail == "" {
		return mailNotifier.processErr("base mail")
	}

	if basePort == "" {
		return mailNotifier.processErr("base port")
	}

	if pwd == "" {
		return mailNotifier.processErr("pwd")
	}

	mailNotifier.host = host
	mailNotifier.baseMail = baseMail
	mailNotifier.basePort = basePort
	mailNotifier.pwd = pwd

	return mailNotifier, nil
}

func (m *MailNotifier) SetNextNotifier(notifier cases.Notifier) {
	slog.Info("Setting in mail notifier next notifier")
	m.next = notifier
}

func (m *MailNotifier) Next() cases.Notifier {
	if m.next == nil {
		slog.Info("No next notifier set for mail notifier")
		return nil
	}
	return m.next
}

func (m *MailNotifier) Notify(sessionResult *entities.SessionResult,
	recipient *entities.Recipient) error {
	slog.Info("Notify for mail notifier started")

	to := m.checkMailInContacts(recipient)
	if to == "" {
		slog.Warn("Recipient mail address not found")
		if nextNotifier := m.Next(); nextNotifier != nil {
			return nextNotifier.Notify(sessionResult, recipient)
		}
		slog.Warn("Mail notifier is last. Message not be sent")
		return nil
	}

	var resultStr string
	for question, answers := range sessionResult.UserAnswer {
		answersJoined := strings.Join(answers, ";")
		resultStr += fmt.Sprintf("Вопрос: %s. Ответ пользователя: %s\n", question, answersJoined)
	}

	subject := fmt.Sprintf(TitlePrefix, sessionResult.GetUserID())
	body := fmt.Sprintf("Topics: \n%s\n\n", strings.Join(sessionResult.Topics, ";\n"))
	body += fmt.Sprintf("Answer:\n%s\n\n", strings.TrimSpace(resultStr))
	body += fmt.Sprintf("IsExpired: %t\n\n", sessionResult.IsExpire)
	body += fmt.Sprintf("IsSuccess: %t\n\n", sessionResult.IsSuccess)
	body += fmt.Sprintf("Resume: %s\n", sessionResult.Resume)

	message := fmt.Sprintf("Subject: %s\r\nTo: %s\r\n\r\n%s\r\n", subject, to, body)

	messageBytes := []byte(message)

	auth := smtp.PlainAuth("", m.baseMail, m.pwd, m.host)

	err := smtp.SendMail(m.host+":"+m.basePort, auth, m.baseMail, []string{to}, messageBytes)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to send email: %v", err)
		slog.Error(err.Error())
		if next := m.Next(); next != nil {
			return next.Notify(sessionResult, recipient)
		}
		return err
	}
	fmt.Println("Email sent successfully!")

	slog.Info("notification by email sent successfully")
	return nil
}

func (m *MailNotifier) processErr(failureArg string) (*MailNotifier, error) {
	err := errors.Wrapf(entities.ErrInvalidParam, "%s is invalid", failureArg)
	slog.Error(err.Error())
	return nil, err
}

func (m *MailNotifier) checkMailInContacts(recipient *entities.Recipient) string {
	slog.Info("Checking mail in contacts started")

	contacts := []string{"mail", "email", "e-mail", "почта", "электронная почта", "почтовый ящик"}

	for _, probableContact := range contacts {
		recipientMailAddress, ok := recipient.Contacts[probableContact]
		if ok {
			slog.Info("Checking mail in contacts finished, mail found")
			return recipientMailAddress
		}
	}

	slog.Info("Checking mail in contacts finished, contacts not found")
	return ""
}
