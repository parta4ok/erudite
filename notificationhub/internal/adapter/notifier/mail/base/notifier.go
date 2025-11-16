package base

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/smtp"
	"net/textproto"

	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/pkg/errors"
)

const (
	TitlePrefix = "Session result for student: %s"
)

var (
	_ cases.Notifier = (*MailNotifier)(nil)
)

var (
	emailTitles = [...]entities.DeliveryService{
		"mail",
		"email",
		"e-mail",
		"почта",
		"электронная почта",
		"почтовый ящик",
	}
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

func (m *MailNotifier) Notify(ctx context.Context, message entities.Event) error {
	slog.Info("Notify for mail notifier started")

	email := m.checkMailInContacts(message.GetRecipient().Contacts)
	if email == "" {
		slog.Warn("Recipient mail address not found")
		if nextNotifier := m.Next(); nextNotifier != nil {
			return nextNotifier.Notify(ctx, message)
		}
		slog.Warn("Mail notifier is last. Message not be sent")
		return nil
	}

	auth := smtp.PlainAuth("", m.baseMail, m.pwd, m.host)

	body, err := m.createEmailWithAttachment(message, email.String())
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to create email body: %v", err)
		slog.Error(err.Error())
		if next := m.Next(); next != nil {
			return next.Notify(ctx, message)
		}
		return err
	}

	err = smtp.SendMail(
		fmt.Sprintf("%s:%s", m.host, m.basePort),
		auth,
		m.baseMail,
		[]string{email.String()},
		body,
	)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to send email: %v", err)
		slog.Error(err.Error())
		if next := m.Next(); next != nil {
			return next.Notify(ctx, message)
		}
		return err
	}

	slog.Info("notification by email sent successfully")
	return nil
}

//nolint:gosec //ok
func (m *MailNotifier) createEmailWithAttachment(message entities.Event, recipientEmail string,
) ([]byte, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	header := make(textproto.MIMEHeader)
	header.Set("From", m.baseMail)
	header.Set("To", recipientEmail)
	header.Set("Subject", message.Kind().String())
	header.Set("MIME-Version", "1.0")
	header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()))

	for key, values := range header {
		for _, value := range values {
			buffer.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}
	buffer.WriteString("\r\n")

	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=UTF-8"},
	})
	if err != nil {
		return nil, err
	}
	textPart.Write([]byte("Please find the report attached.")) //nolint:errcheck //ok

	fileName := fmt.Sprintf("%s.%s", message.Kind(), message.Format())
	attachmentHeader := make(textproto.MIMEHeader)
	attachmentHeader.Set("Content-Type", "application/octet-stream")
	attachmentHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	attachmentHeader.Set("Content-Transfer-Encoding", "base64")

	attachmentPart, err := writer.CreatePart(attachmentHeader)
	if err != nil {
		return nil, err
	}

	encoder := base64.NewEncoder(base64.StdEncoding, attachmentPart)
	encoder.Write(message.Payload()) //nolint:errcheck //ok
	encoder.Close()                  //nolint:errcheck //ok

	writer.Close() //nolint:errcheck //ok
	return buffer.Bytes(), nil
}

func (m *MailNotifier) processErr(failureArg string) (*MailNotifier, error) {
	err := errors.Wrapf(entities.ErrInvalidParam, "%s is invalid", failureArg)
	slog.Error(err.Error())
	return nil, err
}

func (m *MailNotifier) checkMailInContacts(
	contacts map[entities.DeliveryService]entities.Contact) entities.Contact {
	slog.Info("Checking mail in contacts started")

	for _, probableContact := range emailTitles {
		address, ok := contacts[probableContact]
		if ok {
			slog.Info("Checking mail in contacts finished, mail found")
			return address
		}
	}

	slog.Info("Checking mail in contacts finished, contacts not found")
	return ""
}
